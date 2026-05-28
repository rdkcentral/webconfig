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
	"encoding/json"
	"fmt"
	"time"
)

const (
	VersionChanged = iota + 1
	BitmapChanged
	VersionAndBitmapChanged
)

type RootDocument struct {
	Bitmap          int    `json:"bitmap"`
	FirmwareVersion string `json:"firmware_version"`
	ModelName       string `json:"model_name"`
	PartnerId       string `json:"partner_id"`
	SchemaVersion   string `json:"schema_version"`
	Version         string `json:"version"`
	QueryParams     string `json:"query_params"`
	LockedTill      int    `json:"locked_till"`
	ProductClass    string `json:"product_class"`
	AccountType     string `json:"account_type"`
}

// (bitmap, firmware_version, model_name, partner_id, schema_version, version, query_params, product_class, account_type), nil
func NewRootDocument(bitmap int, firmwareVersion, modelName, partnerId, schemaVersion, version, query_params, productClass, accountType string) *RootDocument {
	return &RootDocument{
		Bitmap:          bitmap,
		FirmwareVersion: firmwareVersion,
		ModelName:       modelName,
		PartnerId:       partnerId,
		SchemaVersion:   schemaVersion,
		Version:         version,
		QueryParams:     query_params,
		ProductClass:    productClass,
		AccountType:     accountType,
	}
}

func (d *RootDocument) ColumnMap() map[string]interface{} {
	dict := map[string]interface{}{
		"bitmap":           d.Bitmap,
		"firmware_version": d.FirmwareVersion,
		"model_name":       d.ModelName,
		"partner_id":       d.PartnerId,
		"schema_version":   d.SchemaVersion,
		"version":          d.Version,
		"query_params":     d.QueryParams,
		"product_class":    d.ProductClass,
		"account_type":     d.AccountType,
	}
	return dict
}

func (d *RootDocument) NonEmptyColumnMap() map[string]interface{} {
	dict := make(map[string]interface{})
	if d.Bitmap > 0 {
		dict["bitmap"] = d.Bitmap
	}
	if d.LockedTill > 0 {
		dict["locked_till"] = int64(d.LockedTill)
	}

	tempDict := map[string]string{
		"firmware_version": d.FirmwareVersion,
		"model_name":       d.ModelName,
		"partner_id":       d.PartnerId,
		"schema_version":   d.SchemaVersion,
		"version":          d.Version,
		"query_params":     d.QueryParams,
		"product_class":    d.ProductClass,
		"account_type":     d.AccountType,
	}

	for k, v := range tempDict {
		if len(v) > 0 {
			dict[k] = v
		}
	}
	return dict
}

func (d *RootDocument) Compare(r *RootDocument) int {
	if d.Bitmap != r.Bitmap {
		return RootDocumentMetaChanged
	}
	if d.FirmwareVersion != r.FirmwareVersion {
		return RootDocumentMetaChanged
	}
	if d.ModelName != r.ModelName {
		return RootDocumentMetaChanged
	}
	// only real non-empty differences is considered changed
	if len(d.PartnerId) > 0 && len(r.PartnerId) > 0 && d.PartnerId != r.PartnerId {
		return RootDocumentMetaChanged
	}
	if d.SchemaVersion != r.SchemaVersion {
		return RootDocumentMetaChanged
	}
	if d.ProductClass != r.ProductClass {
		return RootDocumentMetaChanged
	}
	if d.AccountType != r.AccountType {
		return RootDocumentMetaChanged
	}
	if d.Version != r.Version {
		return RootDocumentVersionOnlyChanged
	}
	if len(d.Version) == 0 {
		return RootDocumentMissing
	}
	return RootDocumentEquals
}

func (d *RootDocument) Equals(r *RootDocument) bool {
	if d.Bitmap != r.Bitmap {
		return false
	}
	if d.FirmwareVersion != r.FirmwareVersion {
		return false
	}
	if d.ModelName != r.ModelName {
		return false
	}
	if d.PartnerId != r.PartnerId {
		return false
	}
	if d.SchemaVersion != r.SchemaVersion {
		return false
	}
	if d.ProductClass != r.ProductClass {
		return false
	}
	if d.AccountType != r.AccountType {
		return false
	}
	return true
}

// update in place
func (d *RootDocument) Update(r *RootDocument) {
	if r.Bitmap > 0 {
		d.Bitmap = r.Bitmap
	}
	if len(r.FirmwareVersion) > 0 {
		d.FirmwareVersion = r.FirmwareVersion
	}
	if len(r.ModelName) > 0 {
		d.ModelName = r.ModelName
	}
	if len(r.PartnerId) > 0 {
		d.PartnerId = r.PartnerId
	}
	if len(r.SchemaVersion) > 0 {
		d.SchemaVersion = r.SchemaVersion
	}
	if len(r.Version) > 0 {
		d.Version = r.Version
	}
	if len(r.QueryParams) > 0 {
		d.QueryParams = r.QueryParams
	}
	if len(r.ProductClass) > 0 {
		d.ProductClass = r.ProductClass
	}
	if len(r.AccountType) > 0 {
		d.AccountType = r.AccountType
	}
}

func (d *RootDocument) UpdateMetadata(r *RootDocument) {
	// Version and QueryParams are cloud data, so not changed
	if r.Bitmap > 0 {
		d.Bitmap = r.Bitmap
	}
	if len(r.FirmwareVersion) > 0 {
		d.FirmwareVersion = r.FirmwareVersion
	}
	if len(r.ModelName) > 0 {
		d.ModelName = r.ModelName
	}
	if len(r.PartnerId) > 0 {
		d.PartnerId = r.PartnerId
	}
	if len(r.SchemaVersion) > 0 {
		d.SchemaVersion = r.SchemaVersion
	}
	if len(r.ProductClass) > 0 {
		d.ProductClass = r.ProductClass
	}
	if len(r.AccountType) > 0 {
		d.AccountType = r.AccountType
	}
}

func (d *RootDocument) String() string {
	m := d.ColumnMap()
	bbytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", m)
	}
	return string(bbytes)
}

func (d *RootDocument) Clone() *RootDocument {
	obj := *d
	return &obj
}

func (d *RootDocument) Locked() bool {
	return d.LockedTill > 0 && int(time.Now().UnixMilli()) < d.LockedTill
}
