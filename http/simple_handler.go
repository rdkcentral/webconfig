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
package http

import (
	"fmt"
	"net/http"

	"github.com/rdkcentral/webconfig/common"
)

func (s *WebconfigServer) VersionHandler(w http.ResponseWriter, r *http.Request) {
	version := common.Version{
		CodeGitCommit:   s.GetString("webconfig.code_git_commit"),
		BuildTime:       s.GetString("webconfig.build_time"),
		BinaryVersion:   common.BinaryVersion,
		BinaryBranch:    common.BinaryBranch,
		BinaryBuildTime: common.BinaryBuildTime,
		OpenLibVersion:  common.OpenLibVersion,
	}
	WriteOkResponse(w, version)
}

func (s *WebconfigServer) MonitorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-length", "0")
}

func (s *WebconfigServer) NotificationHandler(w http.ResponseWriter, r *http.Request) {
	_, err := getValue()
	if err != nil {
		Error(w, http.StatusInternalServerError, common.NewError(err))
		return
	}
	WriteOkResponse(w, nil)
}

func (s *WebconfigServer) ServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(s.ConfigBytes())
}

func getValue() (string, error) {
	err := fmt.Errorf("hello world ERR")
	return "", common.NewError(err)
}
