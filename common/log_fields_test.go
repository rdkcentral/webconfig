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
		"red":             "maroon",
		"orange":          "auburn",
		"yellow":          "amber",
		"green":           "viridian",
		"blue":            "turquoise",
		"indigo":          "sapphire",
		"violet":          "purple",
		"out_traceparent": "foo",
		"out_tracestate":  "cyan",
		"token":           "bar",
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
