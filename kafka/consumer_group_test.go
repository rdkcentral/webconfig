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
	"testing"
	"time"

	"github.com/IBM/sarama"
	"gotest.tools/assert"
)

func TestGetEventName(t *testing.T) {
	// ==== mqtt-get ====
	rheader := &sarama.RecordHeader{
		Key:   []byte("rpt"),
		Value: []byte("x/fr/webconfig/get"),
	}
	headers := []*sarama.RecordHeader{
		rheader,
	}
	m := &sarama.ConsumerMessage{
		Topic:     "topic1",
		Partition: int32(1),
		Key:       []byte("hello"),
		Value:     []byte("world"),
		Offset:    int64(1),
		Timestamp: time.Now(),
		Headers:   headers,
	}
	eventName, rptHeaderValue := getEventName(m)
	assert.Equal(t, eventName, "mqtt-get")
	assert.Equal(t, rptHeaderValue, "x/fr/webconfig/get")

	// ==== mqtt-state ====
	rheader = &sarama.RecordHeader{
		Key:   []byte("rpt"),
		Value: []byte("x/fr/webconfig/poke"),
	}
	headers = []*sarama.RecordHeader{
		rheader,
	}
	m = &sarama.ConsumerMessage{
		Topic:     "topic2",
		Partition: int32(2),
		Key:       []byte("foo"),
		Value:     []byte("bar"),
		Offset:    int64(2),
		Timestamp: time.Now(),
		Headers:   headers,
	}
	eventName, rptHeaderValue = getEventName(m)
	assert.Equal(t, eventName, "mqtt-state")
	assert.Equal(t, rptHeaderValue, "x/fr/webconfig/poke")

	// ==== webpa-state ====
	m = &sarama.ConsumerMessage{
		Topic:     "topic3",
		Partition: int32(3),
		Key:       []byte("red"),
		Value:     []byte("orange"),
		Offset:    int64(3),
		Timestamp: time.Now(),
	}
	eventName, rptHeaderValue = getEventName(m)
	assert.Equal(t, eventName, "webpa-state")
	assert.Equal(t, rptHeaderValue, "")

	// ==== unknown-no-rpt ====
	rheader = &sarama.RecordHeader{
		Key:   []byte("yellow"),
		Value: []byte("green"),
	}
	headers = []*sarama.RecordHeader{
		rheader,
	}
	m = &sarama.ConsumerMessage{
		Topic:     "topic4",
		Partition: int32(4),
		Key:       []byte("blue"),
		Value:     []byte("indigo"),
		Offset:    int64(4),
		Timestamp: time.Now(),
		Headers:   headers,
	}
	eventName, rptHeaderValue = getEventName(m)
	assert.Equal(t, eventName, "unknown-no-rpt")
	assert.Equal(t, rptHeaderValue, "")

	// ==== unknown-rpt ====
	rheader1 := &sarama.RecordHeader{
		Key:   []byte("yellow"),
		Value: []byte("green"),
	}
	rheader2 := &sarama.RecordHeader{
		Key:   []byte("rpt"),
		Value: []byte("indigo"),
	}
	headers = []*sarama.RecordHeader{
		rheader1,
		rheader2,
	}
	m = &sarama.ConsumerMessage{
		Topic:     "topic4",
		Partition: int32(4),
		Key:       []byte("blue"),
		Value:     []byte("indigo"),
		Offset:    int64(4),
		Timestamp: time.Now(),
		Headers:   headers,
	}
	eventName, rptHeaderValue = getEventName(m)
	assert.Equal(t, eventName, "unknown-rpt")
	assert.Equal(t, rptHeaderValue, "indigo")
}
