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
	"bytes"
)

type RefSubDocument struct {
	payload []byte
	version *string
}

func NewRefSubDocument(payload []byte, version *string) *RefSubDocument {
	return &RefSubDocument{
		payload: payload,
		version: version,
	}
}

func (d *RefSubDocument) Payload() []byte {
	return d.payload
}

func (d *RefSubDocument) SetPayload(payload []byte) {
	d.payload = payload
}

func (d *RefSubDocument) HasPayload() bool {
	if d.payload != nil && len(d.payload) > 0 {
		return true
	} else {
		return false
	}
}

func (d *RefSubDocument) Version() *string {
	return d.version
}

func (d *RefSubDocument) SetVersion(version *string) {
	d.version = version
}

func (d *RefSubDocument) Equals(tdoc *RefSubDocument) bool {
	if d.HasPayload() && tdoc.HasPayload() {
		if !bytes.Equal(d.Payload(), tdoc.Payload()) {
			return false
		}
	} else {
		if d.HasPayload() != tdoc.HasPayload() {
			return false
		}
	}

	if d.Version() != nil && tdoc.Version() != nil {
		if *d.Version() != *tdoc.Version() {
			return false
		}
	} else {
		if d.Version() != tdoc.Version() {
			return false
		}
	}
	return true
}
