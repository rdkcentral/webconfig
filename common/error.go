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
	"fmt"
	"path/filepath"
	"runtime"
)

var (
	NotOK           = fmt.Errorf("!ok")
	ProfileNotFound = fmt.Errorf("profile not found")
)

type Http400Error struct {
	Message string
}

func (e Http400Error) Error() string {
	return e.Message
}

func NewHttp400Error(message string) *Http400Error {
	return &Http400Error{
		Message: message,
	}
}

type Http404Error struct {
	Message string
}

func (e Http404Error) Error() string {
	return e.Message
}

func NewHttp404Error(message string) *Http404Error {
	return &Http404Error{
		Message: message,
	}
}

type Http500Error struct {
	Message string
}

func (e Http500Error) Error() string {
	return e.Message
}

func NewHttp500Error(message string) *Http500Error {
	return &Http500Error{
		Message: message,
	}
}

type RemoteHttpError struct {
	StatusCode int
	Message    string
}

func (e RemoteHttpError) Error() string {
	return fmt.Sprintf("Http%v %v", e.StatusCode, e.Message)
}

type GroupVersionMismatchError struct {
	message string
}

func (e GroupVersionMismatchError) Error() string {
	return e.message
}

func NewGroupVersionMismatchError(message string) *GroupVersionMismatchError {
	return &GroupVersionMismatchError{
		message: message,
	}
}

var (
	Http400ErrorType    = &Http400Error{}
	Http404ErrorType    = &Http404Error{}
	Http500ErrorType    = &Http500Error{}
	RemoteHttpErrorType = &RemoteHttpError{}
)

func NewError(err error) error {
	_, file, line, _ := runtime.Caller(1)
	filename := filepath.Base(file)
	fulldir := filepath.Dir(file)
	dir := filepath.Base(fulldir)
	return fmt.Errorf("%v/%v[%v]  %w", dir, filename, line, err)
}

func UnwrapAll(wrappedErr error) error {
	err := wrappedErr
	for i := 0; i < 10; i++ {
		unerr := errors.Unwrap(err)
		if unerr == nil {
			return err
		}
		err = unerr
	}
	return err
}

func GetCaller() string {
	_, file, line, _ := runtime.Caller(1)
	filename := filepath.Base(file)
	fulldir := filepath.Dir(file)
	dir := filepath.Base(fulldir)
	return fmt.Sprintf("%v/%v[%v]", dir, filename, line)
}
