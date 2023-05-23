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
	"fmt"
)

type SubDocument struct {
	payload      []byte
	version      *string
	state        *int
	updatedTime  *int
	errorCode    *int
	errorDetails *string
	expiry       *int
}

func NewSubDocument(payload []byte, version *string, state *int, updatedTime *int, errorCode *int, errorDetails *string) *SubDocument {
	return &SubDocument{
		payload:      payload,
		version:      version,
		state:        state,
		updatedTime:  updatedTime,
		errorCode:    errorCode,
		errorDetails: errorDetails,
	}
}

func (d SubDocument) String() string {
	var s1, s2, s3, s4, s5, s6, s7 string
	if d.payload == nil {
		s1 = "payload=nil"
	} else {
		s1 = fmt.Sprintf("len(payload)=%v", len(d.payload))
	}
	if d.version == nil {
		s3 = "version=nil"
	} else {
		s3 = fmt.Sprintf("version=%v", *d.version)
	}
	if d.state == nil {
		s4 = "state=nil"
	} else {
		s4 = fmt.Sprintf("state=%v", *d.state)
	}
	if d.updatedTime == nil {
		s5 = "updatedTime=nil"
	} else {
		s5 = fmt.Sprintf("updatedTime=%v", *d.updatedTime)
	}
	if d.errorCode == nil {
		s6 = "errorCode=nil"
	} else {
		s6 = fmt.Sprintf("errorCode=%v", *d.errorCode)
	}
	if d.errorDetails == nil {
		s7 = "errorDetails=nil"
	} else {
		s7 = fmt.Sprintf("errorDetails=%v", *d.errorDetails)
	}
	return fmt.Sprintf("SubDocument(%v, %v, %v, %v, %v, %v, %v)", s1, s2, s3, s4, s5, s6, s7)
}

func (d *SubDocument) Payload() []byte {
	return d.payload
}

func (d *SubDocument) SetPayload(payload []byte) {
	d.payload = payload
}

func (d *SubDocument) HasPayload() bool {
	if d.payload != nil && len(d.payload) > 0 {
		return true
	} else {
		return false
	}
}

func (d *SubDocument) Version() *string {
	return d.version
}

func (d *SubDocument) SetVersion(version *string) {
	d.version = version
}

func (d *SubDocument) State() *int {
	return d.state
}

func (d *SubDocument) SetState(state *int) {
	d.state = state
}

func (d *SubDocument) UpdatedTime() *int {
	return d.updatedTime
}

func (d *SubDocument) SetUpdatedTime(updatedTime *int) {
	d.updatedTime = updatedTime
}

func (d *SubDocument) ErrorCode() *int {
	return d.errorCode
}

func (d *SubDocument) SetErrorCode(errorCode *int) {
	d.errorCode = errorCode
}

func (d *SubDocument) ErrorDetails() *string {
	return d.errorDetails
}

func (d *SubDocument) SetErrorDetails(errorDetails *string) {
	d.errorDetails = errorDetails
}

func (d *SubDocument) Expiry() *int {
	return d.expiry
}

func (d *SubDocument) SetExpiry(expiry *int) {
	d.expiry = expiry
}

func (d *SubDocument) Equals(tdoc *SubDocument) error {
	if d.HasPayload() && tdoc.HasPayload() {
		if !bytes.Equal(d.Payload(), tdoc.Payload()) {
			err := fmt.Errorf("d.Payload() != tdoc.Payload(), len(d.Payload())=%v, len(tdoc.Payload())=%v", len(d.payload), len(tdoc.payload))
			return NewError(err)
		}
	} else {
		if d.HasPayload() != tdoc.HasPayload() {
			err := fmt.Errorf("d.HasPayload() != tdoc.HasPayload()")
			return NewError(err)
		}
	}

	if d.Version() != nil && tdoc.Version() != nil {
		if *d.Version() != *tdoc.Version() {
			err := fmt.Errorf("*d.Version()[%v] != *tdoc.Version()[%v]", *d.Version(), *tdoc.Version())
			return NewError(err)
		}
	} else {
		if d.Version() != tdoc.Version() {
			err := fmt.Errorf("d.Version()[%v] != tdoc.Version()[%v]", d.Version(), tdoc.Version())
			return NewError(err)
		}
	}

	if d.UpdatedTime() != nil && tdoc.UpdatedTime() != nil {
		if *d.UpdatedTime() != *tdoc.UpdatedTime() {
			err := fmt.Errorf("*d.UpdatedTime()[%v] != *tdoc.UpdatedTime()[%v]", *d.UpdatedTime(), *tdoc.UpdatedTime())
			return NewError(err)
		}
	} else {
		if d.UpdatedTime() != tdoc.UpdatedTime() {
			err := fmt.Errorf("d.UpdatedTime()[%v] != tdoc.UpdatedTime()[%v]", d.UpdatedTime(), tdoc.UpdatedTime())
			return NewError(err)
		}
	}

	if d.State() != nil && tdoc.State() != nil {
		if *d.State() != *tdoc.State() {
			err := fmt.Errorf("*d.State()[%v] != *tdoc.State()[%v]", *d.State(), *tdoc.State())
			return NewError(err)
		}
	} else {
		if d.State() != tdoc.State() {
			err := fmt.Errorf("d.State()[%v] != tdoc.State()[%v]", d.State(), tdoc.State())
			return NewError(err)
		}
	}

	if d.ErrorCode() != nil && tdoc.ErrorCode() != nil {
		if *d.ErrorCode() != *tdoc.ErrorCode() {
			err := fmt.Errorf("*d.ErrorCode()[%v] != *tdoc.ErrorCode()[%v]", *d.ErrorCode(), *tdoc.ErrorCode())
			return NewError(err)
		}
	} else {
		var dErrorCode, tdocErrorCode int
		if d.ErrorCode() != nil {
			dErrorCode = *d.ErrorCode()
		}
		if tdoc.ErrorCode() != nil {
			tdocErrorCode = *tdoc.ErrorCode()
		}
		if dErrorCode != tdocErrorCode {
			err := fmt.Errorf("d.ErrorCode()[%v] != tdoc.ErrorCode()[%v]", d.ErrorCode(), tdoc.ErrorCode())
			return NewError(err)
		}
	}

	if d.ErrorDetails() != nil && tdoc.ErrorDetails() != nil {
		if *d.ErrorDetails() != *tdoc.ErrorDetails() {
			err := fmt.Errorf("*d.ErrorDetails()[%v] != *tdoc.ErrorDetails()[%v]", *d.ErrorDetails(), *tdoc.ErrorDetails())
			return NewError(err)
		}
	} else {
		var dErrorDetails, tdocErrorDetails string
		if d.ErrorDetails() != nil {
			dErrorDetails = *d.ErrorDetails()
		}
		if tdoc.ErrorDetails() != nil {
			tdocErrorDetails = *tdoc.ErrorDetails()
		}
		if dErrorDetails != tdocErrorDetails {
			err := fmt.Errorf("d.ErrorDetails()[%v] != tdoc.ErrorDetails()[%v]", d.ErrorDetails(), tdoc.ErrorDetails())
			return NewError(err)
		}
	}

	if d.Expiry() != nil && tdoc.Expiry() != nil {
		if *d.Expiry() != *tdoc.Expiry() {
			err := fmt.Errorf("*d.Expiry()[%v] != *tdoc.Expiry()[%v]", *d.Expiry(), *tdoc.Expiry())
			return NewError(err)
		}
	} else {
		if d.Expiry() != tdoc.Expiry() {
			err := fmt.Errorf("d.Expiry()[%v] != tdoc.Expiry()[%v]", d.Expiry(), tdoc.Expiry())
			return NewError(err)
		}
	}

	return nil
}

func (d *SubDocument) NeedsUpdateForHttp304() bool {
	if d.updatedTime != nil && *d.updatedTime < 0 {
		return true
	}

	if d.state != nil {
		if *d.state != Deployed {
			return true
		}
		// if state == Deployed
		if d.errorCode != nil && *d.errorCode != 0 {
			return true
		}
		if d.errorDetails != nil && len(*d.errorDetails) > 0 {
			return true
		}
	}
	return false
}
