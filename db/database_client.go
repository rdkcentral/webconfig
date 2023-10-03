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
package db

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rdkcentral/webconfig/common"
)

type DatabaseClient interface {
	SetUp() error
	TearDown() error

	// SubDocument and Document
	GetSubDocument(string, string) (*common.SubDocument, error)
	SetSubDocument(string, string, *common.SubDocument, ...interface{}) error
	DeleteSubDocument(string, string) error

	GetDocument(string, ...interface{}) (*common.Document, error)
	SetDocument(string, *common.Document) error
	DeleteDocument(string) error

	// root document
	GetRootDocument(string) (*common.RootDocument, error)
	SetRootDocument(string, *common.RootDocument) error
	DeleteRootDocument(string) error
	SetRootDocumentVersion(string, string) error
	SetRootDocumentBitmap(string, int) error
	DeleteRootDocumentVersion(string) error
	GetRootDocumentLabels(string) (prometheus.Labels, error)

	// not found
	IsDbNotFound(error) bool

	// set metrics
	Metrics() *common.AppMetrics
	SetMetrics(*common.AppMetrics)

	// blockedSubdocIds
	BlockedSubdocIds() []string
	SetBlockedSubdocIds([]string)

	// These functions are now changed to use upstream
	FactoryReset(string) error
	FirmwareUpdate(string, int, *common.RootDocument) error
	AppendProfiles(string, []byte) ([]byte, error)
}
