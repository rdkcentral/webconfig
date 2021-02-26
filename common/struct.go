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

//http ok response
type HttpResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

//http error response
type HttpErrorResponse struct {
	Status    int         `json:"status"`
	ErrorCode int         `json:"error_code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
}

type Multipart struct {
	Bytes   []byte
	Version string
	Name    string
}

type Version struct {
	CodeGitCommit   string `json:"code_git_commit"`
	BuildTime       string `json:"build_time"`
	BinaryVersion   string `json:"binary_version"`
	BinaryBranch    string `json:"binary_branch"`
	BinaryBuildTime string `json:"binary_build_time"`
}

type SupportedGroupsData struct {
	Bitmap int             `json:"bitmap"`
	Groups map[string]bool `json:"groups"`
}

type SupportedGroupsGetResponse struct {
	Data    SupportedGroupsData `json:"data"`
	Message string              `json:"message"`
	Status  int                 `json:"status"`
}

type PostTokenResponse struct {
	Data    string `json:"data"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}
