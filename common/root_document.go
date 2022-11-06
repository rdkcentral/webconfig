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
}

// (bitmap, firmware_version, model_name, partner_id, schema_version, version), nil
func NewRootDocument(bitmap int, firmwareVersion, modelName, partnerId, schemaVersion, version string) *RootDocument {
	return &RootDocument{
		Bitmap:          bitmap,
		FirmwareVersion: firmwareVersion,
		ModelName:       modelName,
		PartnerId:       partnerId,
		SchemaVersion:   schemaVersion,
		Version:         version,
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
	}
	return dict
}

func (d *RootDocument) NonEmptyColumnMap() map[string]interface{} {
	dict := make(map[string]interface{})
	if d.Bitmap > 0 {
		dict["bitmap"] = d.Bitmap
	}

	tempDict := map[string]string{
		"firmware_version": d.FirmwareVersion,
		"model_name":       d.ModelName,
		"partner_id":       d.PartnerId,
		"schema_version":   d.SchemaVersion,
		"version":          d.Version,
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
	if d.PartnerId != r.PartnerId {
		return RootDocumentMetaChanged
	}
	if d.SchemaVersion != r.SchemaVersion {
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
}
