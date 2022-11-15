/**
* Copyright 2021 Comcast Cable Communications Management, LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
* SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	wchttp "github.com/rdkcentral/webconfig/http"
	"github.com/rdkcentral/webconfig/kafka"
	_ "go.uber.org/automaxprocs"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultConfigFile = "/app/webconfig/webconfig.conf"
)

// main function to boot up everything
func main() {
	mainCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// parse flag
	configFile := flag.String("f", DefaultConfigFile, "config file")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("webconfig version %s (branch %v) %v\n", common.BinaryVersion, common.BinaryBranch, common.BinaryBuildTime)
		os.Exit(0)
	}

	// read new hocon config
	sc, err := common.NewServerConfig(*configFile)
	if err != nil {
		panic(err)
	}
	server := wchttp.NewWebconfigServer(sc, false)

	// setup logging
	logFile := server.GetString("webconfig.log.file")
	if len(logFile) > 0 {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			fmt.Printf("ERROR opening file: %v", err)
			panic(err)
		}
		defer f.Close()
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stdout)
	}

	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: common.LoggingTimeFormat,
		FieldMap: log.FieldMap{
			log.FieldKeyTime: "timestamp",
		},
	})

	// Output to stderr instead of stdout, could also be a file.

	// default log level info
	logLevel := log.InfoLevel
	if parsed, err := log.ParseLevel(server.GetString("webconfig.log.level")); err == nil {
		logLevel = parsed
	}
	log.SetLevel(logLevel)

	// setup router
	router := server.GetRouter(false)

	var metrics *common.AppMetrics
	if server.MetricsEnabled() {
		router.Handle("/metrics", promhttp.Handler())
		metrics = common.NewMetrics()
		server.SetMetrics(metrics)
		handler := metrics.WebMetrics(router)
		server.Handler = handler
	} else {
		server.Handler = router
	}

	// setup contexts groups
	g, gCtx := errgroup.WithContext(mainCtx)

	// setup http server
	g.Go(
		func() error {
			return server.ListenAndServe()
		},
	)

	g.Go(
		func() error {
			<-gCtx.Done()
			fmt.Printf("HTTP server shutdown NOW !!\n")
			return server.Shutdown(context.Background())
		},
	)

	// setup kafka consumer, if config kafka.enabled=false, then kcgroup=nil, err=nil
	kcgroup, err := kafka.NewKafkaConsumerGroup(sc.Config, server, metrics)
	if err != nil {
		panic(err)
	}
	if kcgroup != nil {
		consumer := *(kcgroup.Consumer())
		topics := kcgroup.Topics()

		g.Go(
			func() error {
				for {
					if err := kcgroup.Consume(gCtx, topics, &consumer); err != nil {
						fmt.Printf("kcgroup.Consumer: err=%v\n", err)
						return err
					}
					consumer.Ready = make(chan bool)
				}
			},
		)
		// This is to setup notify AFTER the sarama is running
		// it is more or less optional, without this reading from the chan,
		// the consumer runs anyway.
		<-consumer.Ready

		g.Go(
			func() error {
				<-gCtx.Done()
				fmt.Printf("SARAMA shutdown NOW !!\n")
				return kcgroup.Close()
			},
		)
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("exit reason: %s \n", err)
	}
}
