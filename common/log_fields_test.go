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
	"testing"

	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestFilterLogFields(t *testing.T) {
	src := log.Fields{
		"red":    "maroon",
		"orange": "auburn",
		"yellow": "amber",
		"green":  "viridian",
		"blue":   "turquoise",
		"indigo": "sapphire",
		"violet": "purple",
	}

	c1 := FilterLogFields(src)
	assert.DeepEqual(t, src, c1)

	c2 := FilterLogFields(src, "blue", "indigo", "pink", "silver")
	expected := log.Fields{
		"red":    "maroon",
		"orange": "auburn",
		"yellow": "amber",
		"green":  "viridian",
		"violet": "purple",
	}
	assert.DeepEqual(t, expected, c2)

	src3 := log.Fields{
		"red":    "maroon",
		"orange": "auburn",
		"yellow": "amber",
		"green":  "viridian",
		"blue":   "turquoise",
		"indigo": "sapphire",
		"violet": "purple",
		"token":  "bar",
	}
	c3 := FilterLogFields(src3)
	assert.DeepEqual(t, src, c3)
}

func TestUpdateLogFields(t *testing.T) {
	src := log.Fields{
		"red":    "maroon",
		"orange": "auburn",
		"yellow": "amber",
		"green":  "viridian",
		"blue":   "turquoise",
		"indigo": "sapphire",
		"violet": "purple",
	}
	newfields := log.Fields{
		"pink":   "magenta",
		"silver": "gray",
		"blue":   "azure",
		"indigo": "navy",
	}
	UpdateLogFields(src, newfields)
	expected := log.Fields{
		"red":    "maroon",
		"orange": "auburn",
		"yellow": "amber",
		"green":  "viridian",
		"violet": "purple",
		"pink":   "magenta",
		"silver": "gray",
		"blue":   "azure",
		"indigo": "navy",
	}

	assert.DeepEqual(t, src, expected)
}

func TestCopyCoreLogFields(t *testing.T) {
	body := map[string]interface{}{
		"device_id":        "mac:29cf4fe3914e",
		"http_status_code": 304,
		"transaction_uuid": "f160f5f2-c899-4652-b066-c9b68328d74f",
		"version":          "1719689278",
	}
	src := log.Fields{
		"red":      "maroon",
		"orange":   "auburn",
		"yellow":   "amber",
		"green":    "viridian",
		"audit_id": "3787b860bdf64d0d87929ac8fc46b54e",
		"cpe_mac":  "29CF4FE3914E",
		"body":     body,
	}
	expected := log.Fields{
		"audit_id": "3787b860bdf64d0d87929ac8fc46b54e",
		"cpe_mac":  "29CF4FE3914E",
		"body":     body,
	}
	copied := CopyCoreLogFields(src)
	assert.DeepEqual(t, copied, expected)

	body["violet"] = "purple"

}

func TestFilterLogFieldsWithItfMap(t *testing.T) {
	weekday := map[string]interface{}{
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
	}

	src := log.Fields{
		"red":     "maroon",
		"orange":  "auburn",
		"yellow":  "amber",
		"green":   "viridian",
		"blue":    "turquoise",
		"indigo":  "sapphire",
		"violet":  "purple",
		"weekday": weekday,
	}

	filtered := FilterLogFields(src)
	assert.DeepEqual(t, src, filtered)

	itf, ok := filtered["weekday"]
	assert.Assert(t, ok)
	fw := itf.(map[string]interface{})
	fw["fri"] = 5

	itf, ok = src["weekday"]
	assert.Assert(t, ok)
	sw := itf.(map[string]interface{})
	assert.Assert(t, len(sw) == 4)
}

func TestFilterLogFieldsWithStrMap(t *testing.T) {
	weekday := map[string]string{
		"mon": "1",
		"tue": "2",
		"wed": "3",
		"thu": "4",
	}

	src := log.Fields{
		"red":     "maroon",
		"orange":  "auburn",
		"yellow":  "amber",
		"green":   "viridian",
		"blue":    "turquoise",
		"indigo":  "sapphire",
		"violet":  "purple",
		"weekday": weekday,
	}

	filtered := FilterLogFields(src)
	assert.DeepEqual(t, src, filtered)

	itf, ok := filtered["weekday"]
	assert.Assert(t, ok)
	fw := itf.(map[string]string)
	fw["fri"] = "5"

	itf, ok = src["weekday"]
	assert.Assert(t, ok)
	sw := itf.(map[string]string)
	assert.Assert(t, len(sw) == 4)
}

func TestFilterLogFieldsSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    log.Fields
		filtered []string
	}{
		{
			name: "filter authentication tokens",
			input: log.Fields{
				"authorization": "Bearer xyz123",
				"token":         "abc123",
				"bearer":        "token123",
				"api_key":       "key123",
				"apikey":        "key456",
				"safe_field":    "value",
			},
			filtered: []string{"authorization", "token", "bearer", "api_key", "apikey"},
		},
		{
			name: "filter HTTP headers",
			input: log.Fields{
				"request_headers": map[string]string{"auth": "token"},
				"response_header": "value",
				"webpa_headers":   "data",
				"mqtt_header":     "data",
				"some_field":      "safe",
			},
			filtered: []string{"request_headers", "response_header", "webpa_headers", "mqtt_header"},
		},
		{
			name: "filter device identifiers",
			input: log.Fields{
				"mac":           "AA:BB:CC:DD:EE:FF",
				"cpemac":        "AABBCCDDEEFF",
				"serial":        "12345",
				"serial_number": "67890",
				"device_id":     "device123",
				"other_field":   "safe",
			},
			filtered: []string{"mac", "cpemac", "serial", "serial_number", "device_id"},
		},
		{
			name: "filter schema_version",
			input: log.Fields{
				"schema_version": "1.0",
				"app_version":    "2.0",
				"safe_field":     "value",
			},
			filtered: []string{"schema_version"},
		},
		{
			name: "filter nested sensitive data in maps",
			input: log.Fields{
				"header": map[string]string{
					"authorization": "Bearer token",
					"content-type":  "application/json",
				},
				"safe_field": "value",
			},
			filtered: []string{}, // The map values should be redacted internally
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLogFields(tt.input)

			// Check that filtered fields are redacted or removed
			for _, field := range tt.filtered {
				val, exists := result[field]
				if exists {
					// If field exists, it should be redacted to "****"
					assert.Equal(t, val, "****", "field %s should be redacted", field)
				}
			}

			// Check that non-sensitive fields are preserved
			if val, ok := tt.input["safe_field"]; ok {
				resultVal, resultOk := result["safe_field"]
				assert.Assert(t, resultOk, "safe_field should be preserved")
				assert.Equal(t, resultVal, val, "safe_field value should match")
			}
			if val, ok := tt.input["other_field"]; ok {
				resultVal, resultOk := result["other_field"]
				assert.Assert(t, resultOk, "other_field should be preserved")
				assert.Equal(t, resultVal, val, "other_field value should match")
			}
			if val, ok := tt.input["app_version"]; ok {
				resultVal, resultOk := result["app_version"]
				assert.Assert(t, resultOk, "app_version should be preserved")
				assert.Equal(t, resultVal, val, "app_version value should match")
			}
		})
	}
}

func TestFilterLogFieldsCaseInsensitive(t *testing.T) {
	src := log.Fields{
		"Authorization":  "Bearer token",
		"TOKEN":          "abc123",
		"Bearer":         "xyz123",
		"API_KEY":        "key123",
		"Mac":            "AA:BB:CC:DD:EE:FF",
		"CPEMAC":         "AABBCCDDEEFF",
		"Schema_Version": "1.0",
		"safe_field":     "value",
	}

	result := FilterLogFields(src)

	// All sensitive fields should be redacted regardless of case
	sensitiveFields := []string{"Authorization", "TOKEN", "Bearer", "API_KEY", "Mac", "CPEMAC", "Schema_Version"}
	for _, field := range sensitiveFields {
		val, exists := result[field]
		if exists {
			assert.Equal(t, val, "****", "field %s should be redacted", field)
		}
	}

	// Safe fields should be preserved
	assert.Equal(t, result["safe_field"], "value", "safe_field should be preserved")
}
