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
	"errors"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func Red() error {
	err := RemoteHttpError{
		StatusCode: http.StatusNotFound,
		Message:    "data not found",
	}
	return NewError(err)
}

func Orange() error {
	err := Red()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func Yellow() error {
	err := Orange()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func Green() error {
	err := Yellow()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func Blue() error {
	err := Green()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func Indigo() error {
	err := Blue()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func Violet() error {
	err := Indigo()
	if err != nil {
		return NewError(err)
	}
	return nil
}

func TestUnwrapAll(t *testing.T) {
	err := Violet()
	assert.Assert(t, errors.As(err, RemoteHttpErrorType))
	unerr := UnwrapAll(err)
	rhe, _ := unerr.(RemoteHttpError)
	assert.Equal(t, rhe.StatusCode, http.StatusNotFound)
	assert.Equal(t, rhe.Message, "data not found")
}
