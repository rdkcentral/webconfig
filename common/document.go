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

type Document struct {
	bytes       []byte
	params      *string
	version     *string
	state       *int
	updatedTime *int64
}

func NewDocument(bytes []byte, params *string, version *string, state *int, updatedTime *int64) *Document {
	return &Document{
		bytes:       bytes,
		params:      params,
		version:     version,
		state:       state,
		updatedTime: updatedTime,
	}
}

func (d Document) String() string {
	fmt.Println("Document.String()")
	var s1, s2, s3, s4, s5 string
	if d.bytes == nil {
		s1 = "bytes=nil"
	} else {
		s1 = fmt.Sprintf("bytes=%v", d.bytes)
	}
	if d.params == nil {
		s2 = "params=nil"
	} else {
		s2 = fmt.Sprintf("params=%v", *d.params)
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
	return fmt.Sprintf("%v, %v, %v, %v, %v", s1, s2, s3, s4, s5)
}

func (d *Document) Bytes() []byte {
	return d.bytes
}

func (d *Document) SetBytes(bytes []byte) {
	d.bytes = bytes
}

func (d *Document) HasBytes() bool {
	if d.bytes != nil && len(d.bytes) > 0 {
		return true
	} else {
		return false
	}
}

func (d *Document) Params() *string {
	return d.params
}

func (d *Document) SetParams(params *string) {
	d.params = params
}

func (d *Document) Version() *string {
	return d.version
}

func (d *Document) SetVersion(version *string) {
	d.version = version
}

func (d *Document) State() *int {
	return d.state
}

func (d *Document) SetState(state *int) {
	d.state = state
}

func (d *Document) UpdatedTime() *int64 {
	return d.updatedTime
}

func (d *Document) SetUpdatedTime(updatedTime *int64) {
	d.updatedTime = updatedTime
}

func (d *Document) Equals(tdoc *Document) error {
	if d.HasBytes() && tdoc.HasBytes() {
		if !bytes.Equal(d.Bytes(), tdoc.Bytes()) {
			err := fmt.Errorf("d.Bytes() != tdoc.Bytes()")
			return NewError(err)
		}
	} else {
		if d.HasBytes() != tdoc.HasBytes() {
			err := fmt.Errorf("d.HasBytes() != tdoc.HasBytes()")
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

	if d.Params() != nil && tdoc.Params() != nil {
		if *d.Params() != *tdoc.Params() {
			err := fmt.Errorf("*d.Params()[%v] != *tdoc.Params()[%v]", *d.Params(), *tdoc.Params())
			return NewError(err)
		}
	} else {
		if d.Params() != tdoc.Params() {
			err := fmt.Errorf("d.Params()[%v] != tdoc.Params()[%v]", d.Params(), tdoc.Params())
			return NewError(err)
		}
	}
	return nil
}
