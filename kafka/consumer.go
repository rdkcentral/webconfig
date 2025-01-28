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
	"net/http"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	wchttp "github.com/rdkcentral/webconfig/http"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	*wchttp.WebconfigServer
	Ready                      chan bool
	ratelimitMessagesPerSecond int
	appName                    string
	clusterName                string
	offsetEnum                 int64
	topicPartitionsMap         map[string][]int32
}

func NewConsumer(s *wchttp.WebconfigServer, ratelimitMessagesPerSecond int, m *common.AppMetrics, clusterName string, offsetEnum int64, topicPartitionsMap map[string][]int32) *Consumer {
	return &Consumer{
		WebconfigServer:            s,
		Ready:                      make(chan bool),
		ratelimitMessagesPerSecond: ratelimitMessagesPerSecond,
		appName:                    s.AppName(),
		clusterName:                clusterName,
		offsetEnum:                 offsetEnum,
		topicPartitionsMap:         topicPartitionsMap,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(session sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	if c.topicPartitionsMap != nil {
		for topic, partitions := range c.topicPartitionsMap {
			for _, p := range partitions {
				session.ResetOffset(topic, p, c.offsetEnum, "")
			}
		}
	}
	close(c.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) handleNotification(bbytes []byte, fields log.Fields) (*common.EventMessage, []string, error) {
	var m common.EventMessage
	err := json.Unmarshal(bbytes, &m)
	if err != nil {
		return nil, nil, common.NewError(err)
	}

	fields["body"] = m
	cpeMac, err := m.Validate(true)
	if err != nil {
		return nil, nil, common.NewError(err)
	}

	if m.ErrorDetails != nil && *m.ErrorDetails == "max_retry_reached" {
		return &m, nil, nil
	}

	fields["cpemac"] = cpeMac
	fields["cpe_mac"] = cpeMac
	updatedSubdocIds, err := db.UpdateDocumentState(c.DatabaseClient, cpeMac, &m, fields)
	if err != nil {
		// NOTE return the *eventMessage
		return &m, updatedSubdocIds, common.NewError(err)
	}
	return &m, updatedSubdocIds, nil
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
	fields["cpemac"] = cpeMac
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
	var transactionId string
	if x := rHeader.Get("Transaction-ID"); len(x) > 0 {
		fields["transaction_id"] = x
		fields["trace_id"] = x
		transactionId = x
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
	if len(transactionId) > 0 {
		respHeader.Set("Transaction-ID", transactionId)
	}

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
	// https://github.com/IBM/sarama/blob/master/consumer_group.go#L27-L29
	rl := ratelimit.New(c.ratelimitMessagesPerSecond, ratelimit.WithoutSlack) // per second, no slack.

	for {
		rl.Take()
		select {
		case message := <-claim.Messages():
			if message == nil {
				break
			}

			lag := int(time.Since(message.Timestamp).Nanoseconds() / 1000000)
			start := time.Now()
			auditId := util.GetAuditId()

			kafkaKey := string(message.Key)
			messageLength := len(message.Value)
			fields := log.Fields{
				"logger":          "kafka",
				"app_name":        c.AppName(),
				"kafka_lag":       lag,
				"kafka_key":       kafkaKey,
				"topic":           message.Topic,
				"audit_id":        auditId,
				"cluster_name":    c.ClusterName(),
				"kafka_partition": message.Partition,
				"kafka_offset":    message.Offset,
				"message_length":  messageLength,
			}

			var err error
			logMessage := "discarded"
			var m *common.EventMessage
			var updatedSubdocIds []string

			eventName, rptHeaderValue := getEventName(message)
			switch eventName {
			case "mqtt-get":
				m, err = c.handleGetMessage(message.Value, fields)
				logMessage = "request ends"
			case "mqtt-state":
				header, bbytes := util.ParseHttp(message.Value)
				fields["destination"] = header.Get("Destination")
				m, updatedSubdocIds, err = c.handleNotification(bbytes, fields)
				logMessage = "ok"
			case "webpa-state":
				m, updatedSubdocIds, err = c.handleNotification(message.Value, fields)
				logMessage = "ok"
			}

			session.MarkMessage(message, "")
			duration := int(time.Since(start).Nanoseconds() / 1000000)
			fields["duration"] = duration
			fields["event_name"] = eventName
			fields["rpt"] = rptHeaderValue

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
				metrics.ObserveKafkaLag(eventName, metricsAgent, lag, message.Partition)
				metrics.ObserveKafkaDuration(eventName, metricsAgent, duration)
				status := "success"
				if err != nil {
					status = "fail"
				}
				metrics.CountKafkaEvents(eventName, status, message.Partition)
			}

			if c.KafkaProducerEnabled() && m != nil {
				c.ForwardKafkaMessage(message.Key, m, fields)
				if len(m.Reports) == 0 {
					if m.HttpStatusCode != nil && *m.HttpStatusCode == http.StatusNotModified && len(updatedSubdocIds) > 0 {
						// build a root/success message
						applicationStatus := "success"
						for _, subdocId := range updatedSubdocIds {
							em := &common.EventMessage{
								Namespace:         &subdocId,
								ApplicationStatus: &applicationStatus,
								DeviceId:          m.DeviceId,
								TransactionUuid:   m.TransactionUuid,
								Version:           m.Version,
							}
							c.ForwardKafkaMessage(message.Key, em, fields)
						}
					}
				}
			}
		case <-session.Context().Done():
			return nil
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
