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

	"github.com/Shopify/sarama"
	"github.com/go-akka/configuration"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	wchttp "github.com/rdkcentral/webconfig/http"
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
	rootGroup, err := NewKafkaConsumerGroup(sc.Config, s, m, "root")
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
				case "x/fr/get":
					return "mqtt-get", rptHeaderValue
				case "x/fr/poke":
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
