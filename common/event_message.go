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
package common

import (
	"fmt"
	"strings"
)

type StateReport struct {
	Url              string `json:"url"`
	HttpStatusCode   int    `json:"http_status_code"`
	RequestTimestamp int    `json:"request_timestamp"`
	Version          string `json:"version"`
	TransactionUuid  string `json:"transaction_uuid"`
}

type EventMessage struct {
	Namespace         *string       `json:"namespace,omitempty"`
	ApplicationStatus *string       `json:"application_status,omitempty"`
	ErrorCode         *int          `json:"error_code,omitempty"`
	ErrorDetails      *string       `json:"error_details,omitempty"`
	DeviceId          string        `json:"device_id"`
	HttpStatusCode    *int          `json:"http_status_code,omitempty"`
	TransactionUuid   *string       `json:"transaction_uuid,omitempty"`
	Version           *string       `json:"version,omitempty"`
	Reports           []StateReport `json:"reports,omitempty"`
	MetricsAgent      *string       `json:"metrics_agent,omitempty"`
}

func (m *EventMessage) getCpeMac() string {
	if len(m.DeviceId) == 16 && m.DeviceId[:4] == "mac:" {
		return strings.ToUpper(m.DeviceId[4:])
	}
	return ""
}

func (m *EventMessage) Validate(checkDeviceId bool) (string, error) {
	var cpeMac string
	if checkDeviceId {
		cpeMac = m.getCpeMac()
		if len(cpeMac) == 0 {
			return cpeMac, fmt.Errorf("event without a valid device_id")
		}
	}

	// rootdoc-report
	if m.HttpStatusCode != nil {
		return cpeMac, nil
	}

	// config-version-report
	if len(m.Reports) > 0 {
		return cpeMac, nil
	}

	if m.Namespace != nil && m.ApplicationStatus != nil {
		return cpeMac, nil
	}

	return cpeMac, fmt.Errorf("ill-formatted event")
}

func (m *EventMessage) EventName() string {
	n := "unknown"
	if len(m.Reports) > 0 {
		n = "config-version-report"
	} else if m.HttpStatusCode != nil {
		n = "rootdoc-report"
	} else if m.ApplicationStatus != nil {
		n = "subdoc-report"
	}
	return n
}
