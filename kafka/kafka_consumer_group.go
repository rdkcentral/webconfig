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
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-akka/configuration"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	wchttp "github.com/rdkcentral/webconfig/http"
)

const (
	// Resilience defaults — configurable via webconfig.kafka.* keys.
	defaultMetadataRetryMax        = 10
	defaultMetadataRetryBackoffSec = 2
	defaultNetTimeoutSec           = 10
	defaultSessionTimeoutSec       = 30
)

type KafkaConsumerGroup struct {
	sarama.ConsumerGroup
	db.DatabaseClient
	consumer *Consumer
	topics   []string
}

func NewKafkaConsumerGroup(conf *configuration.Config, s *wchttp.WebconfigServer, m *common.AppMetrics, clusterName string) (*KafkaConsumerGroup, error) {
	var prefix string
	if clusterName == "root" {
		prefix = "webconfig.kafka"
	} else {
		prefix = "webconfig.kafka.clusters." + clusterName
	}

	enabled := conf.GetBoolean(prefix + ".enabled")
	if !enabled {
		return nil, nil
	}

	brokersStr := conf.GetString(prefix + ".brokers")
	if len(brokersStr) == 0 {
		return nil, common.NewError(fmt.Errorf("no brokers in configs"))
	}
	brokers := strings.Split(brokersStr, ",")

	topicsStr := conf.GetString(prefix + ".topics")
	if len(topicsStr) == 0 {
		return nil, common.NewError(fmt.Errorf("no topics in configs"))
	}
	topics := strings.Split(topicsStr, ",")

	group := conf.GetString(prefix + ".consumer_group")
	if conf.GetBoolean(prefix + ".use_random_consumer_group") {
		group = fmt.Sprintf("webconfig_%v", time.Now().Unix())
	}

	assignor := conf.GetString(prefix+".assignor", "roundrobin")
	sconfig := sarama.NewConfig()

	oldest := conf.GetBoolean(prefix + ".oldest")
	newest := conf.GetBoolean(prefix + ".newest")

	var offsetEnum int64
	if newest {
		offsetEnum = sarama.OffsetNewest
		sconfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else if oldest {
		offsetEnum = sarama.OffsetOldest
		sconfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	ratelimitMessagesPerSecond := int(conf.GetInt32(prefix + ".ratelimit.messages_per_second"))

	switch assignor {
	case "sticky":
		sconfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		sconfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		sconfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		return nil, common.NewError(fmt.Errorf("Unrecognized consumer group partition assignor: %s", assignor))
	}

	var topicPartitionsMap map[string][]int32
	var err error
	if newest {
		topicPartitionsMap, err = GetTopicPartitions(brokers, topics, sconfig)
		if err != nil {
			return nil, common.NewError(err)
		}
	}

	// Resilience tuning — survive Kafka broker restarts without restarting the app.
	//
	// Metadata.Retry.Max: lib default 3  → now 10 (configurable)
	//   Extends retry window to 10×2s = 20s, covering the cluster stabilisation
	//   window during leader election after a broker restart.
	sconfig.Metadata.Retry.Max = int(conf.GetInt32(prefix+".metadata.retry_max", defaultMetadataRetryMax))

	// Metadata.Retry.Backoff: lib default 250ms → now 2s (configurable)
	//   Prevents rapid-fire retries against a restarting broker; paced to give
	//   the cluster time to stabilise between attempts.
	sconfig.Metadata.Retry.Backoff = time.Duration(conf.GetInt32(prefix+".metadata.retry_backoff_sec", defaultMetadataRetryBackoffSec)) * time.Second

	// Metadata.RefreshFrequency: lib default 10min → now 5min (configurable)
	//   Proactive background refresh. Catches stale partition leaders (caused by
	//   broker restarts) up to 5min sooner than the library default.
	sconfig.Metadata.RefreshFrequency = time.Duration(conf.GetInt32(prefix+".metadata.refresh_frequency_sec", 300)) * time.Second

	// Consumer.Group.Session.Timeout: lib default 10s → now 30s (configurable)
	//   Time before the broker evicts a consumer whose heartbeats have stopped.
	//   30s gives the consumer group time to survive a restarting group coordinator
	//   without triggering an unnecessary rebalance.
	//
	// Consumer.Group.Heartbeat.Interval: lib default 3s → now Session.Timeout/6 = 5s
	//   The Kafka protocol ceiling is Session.Timeout/3 (10s). Using /6 keeps the
	//   interval at half the ceiling, providing headroom for transient network jitter
	//   during broker restarts without risking false session expiry.
	sessionTimeout := time.Duration(conf.GetInt32(prefix+".consumer.session_timeout_sec", defaultSessionTimeoutSec)) * time.Second
	sconfig.Consumer.Group.Session.Timeout = sessionTimeout
	sconfig.Consumer.Group.Heartbeat.Interval = sessionTimeout / 6

	// Consumer.Retry.Backoff: lib default 2s (configurable)
	//   How long a partition reader waits before retrying a failed fetch.
	sconfig.Consumer.Retry.Backoff = time.Duration(conf.GetInt32(prefix+".consumer.retry_backoff_sec", 2)) * time.Second

	// Net.DialTimeout / ReadTimeout / WriteTimeout: lib default 30s → now 10s (configurable)
	//   Tighter timeouts let the retry loop engage within 10s per attempt instead
	//   of blocking for 30s. Fail-fast on broken broker connections.
	netTimeout := time.Duration(conf.GetInt32(prefix+".net.timeout_sec", defaultNetTimeoutSec)) * time.Second
	sconfig.Net.DialTimeout = netTimeout
	sconfig.Net.ReadTimeout = netTimeout
	sconfig.Net.WriteTimeout = netTimeout

	// Load TLS configuration
	tlsConfig, err := common.LoadKafkaTLSConfig(conf, prefix)
	if err != nil {
		return nil, common.NewError(fmt.Errorf("failed to load TLS configuration for %s: %v", prefix, err))
	}
	if tlsConfig != nil {
		sconfig.Net.TLS.Enable = true
		sconfig.Net.TLS.Config = tlsConfig
	}

	consumer := NewConsumer(s, ratelimitMessagesPerSecond, m, clusterName, offsetEnum, topicPartitionsMap)

	client, err := sarama.NewConsumerGroup(brokers, group, sconfig)
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
	kcgroups := []*KafkaConsumerGroup{}

	rootGroup, err := NewKafkaConsumerGroup(sc.Config, s, m, "root")
	if err != nil {
		return nil, common.NewError(err)
	}
	if rootGroup != nil {
		kcgroups = append(kcgroups, rootGroup)
	}

	clusterNames := sc.KafkaClusterNames()
	for _, clusterName := range clusterNames {
		kcgroup, err := NewKafkaConsumerGroup(sc.Config, s, m, clusterName)
		if err != nil {
			return nil, common.NewError(err)
		}
		if kcgroup == nil {
			continue
		}

		kcgroups = append(kcgroups, kcgroup)
	}
	return kcgroups, nil
}

func getEventName(message *sarama.ConsumerMessage) (string, string) {
	var rptHeaderValue string
	if len(message.Headers) > 0 {
		for _, h := range message.Headers {
			if string(h.Key) == "rpt" {
				rptHeaderValue = string(h.Value)
				switch rptHeaderValue {
				case "x/fr/webconfig/get":
					return "mqtt-get", rptHeaderValue
				case "x/fr/webconfig/poke":
					return "mqtt-state", rptHeaderValue
				}
				return "unknown-rpt", rptHeaderValue
			}
		}
		return "unknown-no-rpt", rptHeaderValue
	}
	return "webpa-state", rptHeaderValue
}

func GetTopicPartitions(brokers, topics []string, config *sarama.Config) (fobj map[string][]int32, ferr error) {
	saramaConsumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, common.NewError(err)
	}
	defer func() {
		if err := saramaConsumer.Close(); err != nil {
			ferr = common.NewError(err)
		}
	}()

	ret := make(map[string][]int32)
	for _, topic := range topics {
		partitions, err := saramaConsumer.Partitions(topic)
		if err != nil {
			return nil, common.NewError(err)
		}
		ret[topic] = partitions
	}
	return ret, nil
}
