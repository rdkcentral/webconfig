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
package kafka

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	wchttp "github.com/rdkcentral/webconfig/http"
	"github.com/rdkcentral/webconfig/util"
	"go.uber.org/ratelimit"
)

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	*wchttp.WebconfigServer
	Ready                      chan bool
	ratelimitMessagesPerSecond int
	appName                    string
	clusterName                string
}

func NewConsumer(s *wchttp.WebconfigServer, ratelimitMessagesPerSecond int, m *common.AppMetrics, clusterName string) *Consumer {
	return &Consumer{
		WebconfigServer:            s,
		Ready:                      make(chan bool),
		ratelimitMessagesPerSecond: ratelimitMessagesPerSecond,
		appName:                    s.AppName(),
		clusterName:                clusterName,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) handleNotification(bbytes []byte, fields log.Fields) (*common.EventMessage, error) {
	var m common.EventMessage
	err := json.Unmarshal(bbytes, &m)
	if err != nil {
		return nil, common.NewError(err)
	}

	fields["body"] = m
	cpeMac, err := m.Validate(true)
	if err != nil {
		return nil, common.NewError(err)
	}

	fields["cpe_mac"] = cpeMac
	err = db.UpdateDocumentState(c.DatabaseClient, cpeMac, &m, fields)
	if err != nil {
		// NOTE return the *eventMessage
		return &m, common.NewError(err)
	}
	return &m, nil
}

// NOTE we choose to return an EventMessage object just to pass along the metricsAgent
func (c *Consumer) handleGetMessage(inbytes []byte, fields log.Fields) (*common.EventMessage, error) {
	rHeader, _ := util.ParseHttp(inbytes)
	params := rHeader.Get(common.HeaderDocName)
	cpeMac := rHeader.Get(common.HeaderDeviceId)
	if len(cpeMac) == 0 {
		cpeMac = rHeader.Get("Mac")
	}
	cpeMac = strings.ToUpper(cpeMac)
	rHeader.Set(common.HeaderDeviceId, cpeMac)

	// TODO parse themis token and extract mac
	fields["cpe_mac"] = cpeMac
	if len(params) > 0 {
		fields["path"] = fmt.Sprintf("/api/v1/device/%v/config?group_id=%v", cpeMac, params)
	} else {
		fields["path"] = fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	}

	var m common.EventMessage
	if x := rHeader.Get(common.HeaderMetricsAgent); len(x) > 0 {
		fields["metrics_agent"] = x
		m.MetricsAgent = &x
	}
	if x := rHeader.Get("Transaction-ID"); len(x) > 0 {
		fields["transaction_id"] = x
	}

	// remote sensitive headers
	logHeaders := rHeader.Clone()
	logHeaders.Del("Authorization")
	d := make(util.Dict)
	d.Update(logHeaders)
	fields["header"] = d
	log.WithFields(fields).Info("request starts")

	// handle empty schema version header
	if x := rHeader.Get(common.HeaderSchemaVersion); len(x) == 0 {
		rHeader.Set(common.HeaderSchemaVersion, "none")
	}

	status, respHeader, respBytes, err := wchttp.BuildWebconfigResponse(c.WebconfigServer, rHeader, common.RouteMqtt, fields)
	if err != nil && respBytes == nil {
		respBytes = []byte(err.Error())
	}

	fields["status"] = status

	mqttBytes := common.BuildPayloadAsHttp(status, respHeader, respBytes)
	_, err = c.PostMqtt(cpeMac, mqttBytes, fields)
	if err != nil {
		return &m, common.NewError(err)
	}
	return &m, nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	rl := ratelimit.New(c.ratelimitMessagesPerSecond, ratelimit.WithoutSlack) // per second, no slack.
	for message := range claim.Messages() {
		rl.Take()
		// log.Printf("%v\n", string(message.Value))
		lag := int(time.Since(message.Timestamp).Nanoseconds() / 1000000)
		start := time.Now()
		auditId := util.GetAuditId()

		fields := log.Fields{
			"logger":       "kafka",
			"app_name":     c.AppName(),
			"kafka_lag":    lag,
			"kafka_key":    message.Key,
			"topic":        message.Topic,
			"audit_id":     auditId,
			"cluster_name": c.ClusterName(),
		}

		var err error
		var logMessage string
		var m *common.EventMessage

		eventName := getEventName(message)
		switch eventName {
		case "mqtt-get":
			m, err = c.handleGetMessage(message.Value, fields)
			logMessage = "request ends"
		case "mqtt-state":
			header, bbytes := util.ParseHttp(message.Value)
			fields["destination"] = header.Get("Destination")
			m, err = c.handleNotification(bbytes, fields)
			logMessage = "ok"
		case "webpa-state":
			m, err = c.handleNotification(message.Value, fields)
			logMessage = "ok"
		}

		session.MarkMessage(message, "")
		duration := int(time.Since(start).Nanoseconds() / 1000000)
		fields["duration"] = duration

		if err != nil {
			if c.IsDbNotFound(err) {
				log.WithFields(fields).Trace("db not found")
			} else {
				fields["error"] = err.Error()
				fields["kafka_message"] = base64.StdEncoding.EncodeToString(message.Value)
				log.WithFields(fields).Error("errors")
			}
		} else {
			log.WithFields(fields).Info(logMessage)
		}

		// build metrics dimensions and update metrics
		metrics := c.WebconfigServer.Metrics()
		if metrics != nil && m != nil {
			metricsAgent := "default"
			if m.MetricsAgent != nil {
				metricsAgent = *m.MetricsAgent
			}
			// TODO try to read metricsAgent from fields["metrics_agent"]
			metrics.ObserveKafkaLag(eventName, metricsAgent, lag)
			metrics.ObserveKafkaDuration(eventName, metricsAgent, duration)
			status := "success"
			if err != nil {
				status = "fail"
			}
			metrics.CountKafkaEvents(eventName, status)
		}
	}
	return nil
}

func (c *Consumer) AppName() string {
	return c.appName
}

func (c *Consumer) ClusterName() string {
	return c.clusterName
}

type KafkaConsumerGroup struct {
	sarama.ConsumerGroup
	db.DatabaseClient
	consumer *Consumer
	topics   []string
}

func NewKafkaConsumerGroup(conf *configuration.Config, s *wchttp.WebconfigServer, m *common.AppMetrics) (*KafkaConsumerGroup, error) {
	enabled := conf.GetBoolean("webconfig.kafka.enabled")
	if !enabled {
		return nil, nil
	}
	brokersStr := conf.GetString("webconfig.kafka.brokers")
	if len(brokersStr) == 0 {
		return nil, common.NewError(fmt.Errorf("no brokers in configs"))
	}
	topicsStr := conf.GetString("webconfig.kafka.topics")
	if len(topicsStr) == 0 {
		return nil, common.NewError(fmt.Errorf("no topics in configs"))
	}
	topics := strings.Split(topicsStr, ",")
	group := conf.GetString("webconfig.kafka.consumer_group")
	if conf.GetBoolean("use_random_consumer_group") {
		group = fmt.Sprintf("webconfig_%v", time.Now().Unix())
	}

	assignor := conf.GetString("webconfig.kafka.assignor", "roundrobin")
	oldest := conf.GetBoolean("webconfig.kafka.oldest")
	ratelimitMessagesPerSecond := int(conf.GetInt32("webconfig.kafka.ratelimit.messages_per_second"))
	sconfig := sarama.NewConfig()

	switch assignor {
	case "sticky":
		sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	case "roundrobin":
		sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	case "range":
		sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	default:
		return nil, common.NewError(fmt.Errorf("Unrecognized consumer group partition assignor: %s", assignor))
	}

	if oldest {
		sconfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumer := NewConsumer(s, ratelimitMessagesPerSecond, m, "root")

	client, err := sarama.NewConsumerGroup(strings.Split(brokersStr, ","), group, sconfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating consumer group client: %v", err)
	}

	return &KafkaConsumerGroup{
		ConsumerGroup:  client,
		DatabaseClient: s.DatabaseClient,
		consumer:       consumer,
		topics:         topics,
	}, nil
}

func (g *KafkaConsumerGroup) Topics() []string {
	return g.topics
}

func (g *KafkaConsumerGroup) Consumer() *Consumer {
	return g.consumer
}

func NewKafkaConsumerGroups(sc *common.ServerConfig, s *wchttp.WebconfigServer, m *common.AppMetrics) ([]*KafkaConsumerGroup, error) {
	rootGroup, err := NewKafkaConsumerGroup(sc.Config, s, m)
	if err != nil {
		return nil, common.NewError(err)
	}

	if rootGroup == nil {
		return nil, nil
	}

	kcgroups := []*KafkaConsumerGroup{
		rootGroup,
	}

	clusterNames := sc.KafkaClusterNames()
	for _, clusterName := range clusterNames {
		prefix := "webconfig.kafka.clusters." + clusterName
		enabled := sc.GetBoolean(prefix + ".enabled")
		if !enabled {
			continue
		}

		brokersStr := sc.GetString(prefix + ".brokers")
		if len(brokersStr) == 0 {
			return nil, common.NewError(fmt.Errorf("no brokers in configs"))
		}
		topicsStr := sc.GetString(prefix + ".topics")
		if len(topicsStr) == 0 {
			return nil, common.NewError(fmt.Errorf("no topics in configs"))
		}
		topics := strings.Split(topicsStr, ",")
		group := sc.GetString(prefix + ".consumer_group")
		if sc.GetBoolean(prefix + ".use_random_consumer_group") {
			group = fmt.Sprintf("webconfig_%v", time.Now().Unix())
		}

		assignor := sc.GetString(prefix+".assignor", "roundrobin")
		oldest := sc.GetBoolean(prefix + ".oldest")
		ratelimitMessagesPerSecond := int(sc.GetInt32(prefix + ".ratelimit.messages_per_second"))
		sconfig := sarama.NewConfig()

		switch assignor {
		case "sticky":
			sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
		case "roundrobin":
			sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
		case "range":
			sconfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
		default:
			return nil, common.NewError(fmt.Errorf("Unrecognized consumer group partition assignor: %s", assignor))
		}

		if oldest {
			sconfig.Consumer.Offsets.Initial = sarama.OffsetOldest
		}

		consumer := NewConsumer(s, ratelimitMessagesPerSecond, m, clusterName)

		client, err := sarama.NewConsumerGroup(strings.Split(brokersStr, ","), group, sconfig)
		if err != nil {
			return nil, fmt.Errorf("Error creating consumer group client: %v", err)
		}

		kcgroup := KafkaConsumerGroup{
			ConsumerGroup:  client,
			DatabaseClient: s.DatabaseClient,
			consumer:       consumer,
			topics:         topics,
		}
		kcgroups = append(kcgroups, &kcgroup)
	}

	return kcgroups, nil
}

func getEventName(message *sarama.ConsumerMessage) string {
	if len(message.Headers) > 0 {
		for _, h := range message.Headers {
			if string(h.Key) == "rpt" {
				switch string(h.Value) {
				case "x/fr/get":
					return "mqtt-get"
				case "x/fr/poke":
					return "mqtt-state"
				}
			}
		}
		return "unknown"
	}
	return "webpa-state"
}
