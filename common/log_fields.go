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
	"maps"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	unloggedFields = []string{
		"moneytrace",
		"token",
		"xpc_trace",
	}
	coreFields = []string{
		"app_name",
		"audit_id",
		"body",
		"cpe_mac",
	}
	nonEmptyFields = []string{
		"traceparent",
		"tracestate",
		"out_traceparent",
		"out_tracestate",
		"req_moracide_tag",
	}
)

func isSensitiveLogKey(k string) bool {
	key := strings.ToLower(k)
	return strings.Contains(key, "passphrase") ||
		strings.Contains(key, "password") ||
		strings.Contains(key, "passwd") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "authorization") ||
		strings.Contains(key, "cookie") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "apikey") ||
		strings.Contains(key, "api_key")
}

func sanitizeLogValue(k string, v interface{}) interface{} {
	if isSensitiveLogKey(k) {
		return "****"
	}

	switch ty := v.(type) {
	case map[string]string:
		redacted := map[string]string{}
		for mk, mv := range ty {
			if isSensitiveLogKey(mk) || k == "header" {
				redacted[mk] = "****"
			} else {
				redacted[mk] = mv
			}
		}
		return redacted
	case map[string]interface{}:
		redacted := map[string]interface{}{}
		for mk, mv := range ty {
			redacted[mk] = sanitizeLogValue(mk, mv)
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(ty))
		for i, iv := range ty {
			redacted[i] = sanitizeLogValue(k, iv)
		}
		return redacted
	default:
		return ty
	}
}

func FilterLogFields(src log.Fields, excludes ...string) log.Fields {
	fields := log.Fields{}
	for k, v := range src {
		switch ty := v.(type) {
		case string:
			if slices.Contains(nonEmptyFields, k) {
				if len(ty) > 0 {
					fields[k] = sanitizeLogValue(k, ty)
				}
			} else {
				fields[k] = sanitizeLogValue(k, ty)
			}
		case map[string]string:
			fields[k] = sanitizeLogValue(k, maps.Clone(ty))
		case map[string]interface{}:
			fields[k] = sanitizeLogValue(k, maps.Clone(ty))
		default:
			fields[k] = sanitizeLogValue(k, ty)
		}
	}

	for _, x := range unloggedFields {
		delete(fields, x)
	}

	if len(excludes) > 0 {
		for _, x := range excludes {
			delete(fields, x)
		}
	}

	return fields
}

func UpdateLogFields(fields, newfields log.Fields) {
	for k, v := range newfields {
		fields[k] = v
	}
}

func CopyCoreLogFields(src log.Fields) log.Fields {
	fields := log.Fields{}
	for _, k := range coreFields {
		if itf, ok := src[k]; ok {
			fields[k] = itf
		}
	}
	return fields
}
