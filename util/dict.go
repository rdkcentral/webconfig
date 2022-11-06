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
package util

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type Dict map[string]interface{}

func (d Dict) TimeToMsecs(field string) {
	v, ok := d[field]
	if ok {
		switch ty := v.(type) {
		case time.Time:
			if ty.IsZero() {
				delete(d, field)
			} else {
				d[field] = int(ty.UnixNano() / 1000000)
			}
		}
	}
}

func (d Dict) MsecsToTime(fields ...string) {
	for _, field := range fields {
		v, ok := d[field]
		if ok {
			var msecs int64
			switch ty := v.(type) {
			case int:
				msecs = int64(ty)
			case float64:
				msecs = int64(ty)
			case int64:
				msecs = int64(ty)
			}
			secs := msecs / 1000
			nsecs := (msecs % 1000) * 1000000
			d[field] = time.Unix(secs, nsecs)
		}
	}
}

func (d Dict) ToInt(fields ...string) {
	for _, field := range fields {
		itf, ok := d[field]
		if ok {
			var v int
			switch ty := itf.(type) {
			case int:
				v = ty
			case float64:
				v = int(ty)
			case int64:
				v = int(ty)
			}
			d[field] = v
		}
	}
}

func (d Dict) ToInt64(fields ...string) {
	for _, field := range fields {
		itf, ok := d[field]
		if ok {
			var v int64
			switch ty := itf.(type) {
			case int:
				v = int64(ty)
			case float64:
				v = int64(ty)
			case int64:
				v = ty
			}
			d[field] = v
		}
	}
}

func (d Dict) String() string {
	return PrettyJson(d)
}

func (d Dict) Copy() Dict {
	newd := Dict{}
	for k, v := range d {
		newd[k] = v
	}
	return newd
}

func CopyLogFields(src log.Fields) log.Fields {
	fields := log.Fields{}
	for k, v := range src {
		fields[k] = v
	}
	return fields
}

func (d Dict) SelectByKeys(names ...string) Dict {
	ndict := Dict{}
	for _, n := range names {
		v, ok := d[n]
		if ok {
			ndict[n] = v
		}
	}
	return ndict
}

func (d Dict) GetString(key string, defaults ...string) string {
	var ret string
	if len(defaults) > 0 {
		ret = defaults[0]
	}

	if itf, ok := d[key]; ok {
		if x, ok := itf.(string); ok {
			ret = x
		}
	}
	return ret
}

func (d Dict) GetNonEmptyString(key string, defaults ...string) string {
	var ret string
	if len(defaults) > 0 {
		ret = defaults[0]
	}

	if itf, ok := d[key]; ok {
		if x, ok := itf.(string); ok {
			if len(x) > 0 {
				ret = x
			}
		}
	}
	return ret
}

func (d Dict) GetBool(key string, defaults ...bool) bool {
	var ret bool
	if len(defaults) > 0 {
		ret = defaults[0]
	}

	if itf, ok := d[key]; ok {
		if x, ok := itf.(bool); ok {
			ret = x
		}
	}
	return ret
}

func (d Dict) GetInt(key string, defaults ...int) int {
	var ret int
	if len(defaults) > 0 {
		ret = defaults[0]
	}

	if itf, ok := d[key]; ok {
		x, ok := itf.(int)
		if ok {
			ret = x
		} else {
			ret = ToInt(itf)
		}
	}
	return ret
}

func ToInt(itf interface{}) int {
	var v int
	switch ty := itf.(type) {
	case int:
		v = ty
	case float64:
		v = int(ty)
	case int64:
		v = int(ty)
	}
	return v
}

func (d Dict) Update(itf interface{}) {
	switch ty := itf.(type) {
	case log.Fields:
		for k, v := range ty {
			d[k] = v
		}
	case Dict:
		for k, v := range ty {
			d[k] = v
		}
	case http.Header:
		for k, v := range ty {
			d[k] = v[0]
		}
	}
}

func HeaderToMap(header http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range header {
		m[k] = v[0]
	}
	return m

}
