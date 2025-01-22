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
