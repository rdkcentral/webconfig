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
	"flag"
	"fmt"
	"os"

	"github.com/rdkcentral/webconfig/common"
	xpchttp "github.com/rdkcentral/webconfig/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultConfigFile = "config/sample_webconfig.conf"
)

// main function to boot up everything
func main() {
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
	server := xpchttp.NewWebconfigServer(sc, false, nil)

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

	if server.MetricsEnabled() {
		router.Handle("/metrics", promhttp.Handler())
		metrics := xpchttp.NewMetrics()
		handler := xpchttp.WebMetrics(metrics, router)
		server.Handler = handler
	} else {
		server.Handler = router
	}

	server.ListenAndServe()
}
