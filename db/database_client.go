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
	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

type DatabaseClient interface {
	SetUp() error
	TearDown() error

	// document and folder
	GetDocument(string, string, log.Fields) (*common.Document, error)
	SetDocument(string, string, *common.Document, log.Fields) error
	GetFolder(string, log.Fields) (*common.Folder, error)
	DeleteDocument(string, string, log.Fields) error
	DeleteFullDocument(string, log.Fields) error

	// root document
	GetRootDocument(string) (*common.RootDocument, error)
	SetRootDocumentVersion(string, string) error
	SetRootDocumentBitmap(string, int) error
	DeleteRootDocument(string) error
	DeleteRootDocumentVersion(string) error

	// not found
	IsDbNotFound(error) bool
}
