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
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
)

// TODO s.MultipartSupplementaryHandler(w, r) should be handled separately
// (1) need to have a dedicate function update states AFTER this function is executed
// (2) read from the existing "root_document" table and build those into the header for upstream
// (3) return a new variable to indicate goUpstream
func BuildGetDocument(c DatabaseClient, rHeader http.Header, route string, fields log.Fields) (*common.Document, *common.RootDocument, *common.RootDocument, map[string]string, bool, error) {
	fieldsDict := make(util.Dict)
	fieldsDict.Update(fields)

	// ==== deviceRootDocument should always be created from request header ====
	var bitmap int
	var err error
	supportedDocs := rHeader.Get(common.HeaderSupportedDocs)
	if len(supportedDocs) > 0 {
		bitmap, err = util.GetCpeBitmap(supportedDocs)
		if err != nil {
			log.WithFields(fields).Warn(common.NewError(err))
		}
	}

	schemaVersion := strings.ToLower(rHeader.Get(common.HeaderSchemaVersion))
	modelName := rHeader.Get(common.HeaderModelName)

	partnerId := rHeader.Get(common.HeaderPartnerID)
	if len(partnerId) == 0 {
		partnerId = fieldsDict.GetString("partner")
	}

	firmwareVersion := rHeader.Get(common.HeaderFirmwareVersion)

	// start with an empty rootDocument.Version, just in case there are errors in parsing the version from headers
	deviceRootDocument := common.NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, "", "")

	// ==== parse mac ====
	mac := rHeader.Get(common.HeaderDeviceId)

	var document *common.Document

	// get version map
	deviceVersionMap, versions, err := parseVersionMap(rHeader)
	if err != nil {
		var gvmErr common.GroupVersionMismatchError
		if errors.As(err, &gvmErr) {
			// log a warning
			log.WithFields(fields).Warn(gvmErr.Error())

			document, err = c.GetDocument(mac, fields)
			if err != nil {
				// TODO what about 404 should be included here
				return nil, nil, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
			}
			deviceVersionMap = RebuildDeviceVersionMap(versions, document.VersionMap())
		} else {
			return nil, nil, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
	}

	// ==== update the deviceRootDocument.Version  ====
	deviceRootDocument.Version = deviceVersionMap["root"]

	// ==== read the cloudRootDocument from db ====
	cloudRootDocument, err := c.GetRootDocument(mac)
	if err != nil {
		if !c.IsDbNotFound(err) {
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
		// no root doc in db, create a new one
		// NOTE need to clone the deviceRootDocument and set the version "" to avoid device root update was set back to cloud
		clonedRootDoc := deviceRootDocument.Clone()
		clonedRootDoc.Version = ""
		if clonedRootDoc.ModelName == "SR213" {
			line := "CREATE schema_version=" + clonedRootDoc.SchemaVersion
			tfields := common.FilterLogFields(fields)
			tfields["logger"] = "rootdoc"
			log.WithFields(tfields).Info(line)
		}
		if err := c.SetRootDocument(mac, clonedRootDoc); err != nil {
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
		// the returned err is dbNotFound
		return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
	}

	// ==== compare if the deviceRootDocument and cloudRootDocument are different ====
	var rootCmpEnum int
	// mget fakes no meta change so that meta are not updated
	if rHeader.Get("User-Agent") == "mget" {
		rootCmpEnum = common.RootDocumentVersionOnlyChanged
	} else {
		rootCmpEnum = cloudRootDocument.Compare(deviceRootDocument)
	}

	if isDiff := cloudRootDocument.IsDifferent(deviceRootDocument); isDiff {
		// need to update rootDoc meta
		// NOTE need to clone the deviceRootDocument and set the version "" to avoid device root update was set back to cloud
		clonedRootDoc := deviceRootDocument.Clone()
		clonedRootDoc.Version = ""
		if clonedRootDoc.ModelName == "SR213" {
			line := "UPDATE schema_version=" + clonedRootDoc.SchemaVersion
			tfields := common.FilterLogFields(fields)
			tfields["logger"] = "rootdoc"
			log.WithFields(tfields).Info(line)
		}
		if err := c.SetRootDocument(mac, clonedRootDoc); err != nil {
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
	}

	switch rootCmpEnum {
	case common.RootDocumentEquals:
		// create an empty "document"
		document := common.NewDocument(cloudRootDocument)
		// no need to update root doc
		return document, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, nil
	case common.RootDocumentVersionOnlyChanged, common.RootDocumentMissing:
		// meta unchanged but subdoc versions change ==> new configs
		// getDoc, then filter
		if document == nil {
			document, err = c.GetDocument(mac, fields)
			if err != nil {
				// 404 should be included here
				return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
			}
		}
		document.SetRootDocument(cloudRootDocument)
		filteredDocument := document.FilterForGet(deviceVersionMap)
		for _, subdocId := range c.BlockedSubdocIds() {
			filteredDocument.DeleteSubDocument(subdocId)
		}
		return filteredDocument, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, nil
	case common.RootDocumentMetaChanged:
		// getDoc, send it upstream
		if document == nil {
			document, err = c.GetDocument(mac, fields)
			if err != nil {
				// 404 should be included here
				return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
			}
		}
		document.SetRootDocument(cloudRootDocument)

		return document, cloudRootDocument, deviceRootDocument, deviceVersionMap, true, nil
	}

	// default, should not come here
	return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, nil
}

func GetValuesStr(length int) string {
	buffer := bytes.NewBufferString("?")
	for i := 0; i < length-1; i++ {
		buffer.WriteString(",?")
	}
	return buffer.String()
}

func GetColumnsStr(columns []string) string {
	buffer := bytes.NewBuffer([]byte{})
	for i, v := range columns {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(v)
	}
	return buffer.String()
}

func GetSetColumnsStr(columns []string) string {
	buffer := bytes.NewBuffer([]byte{})
	for i, c := range columns {
		if i > 0 {
			buffer.WriteString(",")
		}
		s := fmt.Sprintf("%v=?", c)
		buffer.WriteString(s)
	}
	return buffer.String()
}

// deviceVersionMap := parseVersionMap(rHeader, d)
func parseVersionMap(rHeader http.Header) (map[string]string, []string, error) {
	deviceVersionMap := make(map[string]string)

	queryStr := rHeader.Get(common.HeaderDocName)
	subdocIds := strings.Split(queryStr, ",")
	if len(queryStr) == 0 {
		return deviceVersionMap, nil, nil
	}

	ifNoneMatch := rHeader.Get(common.HeaderIfNoneMatch)
	versions := strings.Split(ifNoneMatch, ",")

	if len(subdocIds) != len(versions) {
		msg := fmt.Sprintf("numbers of elements mismatched  group_id=%v  IF-NONE-MATCH=%v", queryStr, ifNoneMatch)
		gvmErr := common.NewGroupVersionMismatchError(msg)
		return nil, versions, common.NewError(*gvmErr)
	}

	for i, subdocId := range subdocIds {
		deviceVersionMap[subdocId] = versions[i]
	}
	return deviceVersionMap, nil, nil
}

func HashRootVersion(itf interface{}) string {
	var versionMap map[string]string
	switch ty := itf.(type) {
	case []common.Multipart:
		versionMap = make(map[string]string)
		for _, mpart := range ty {
			versionMap[mpart.Name] = mpart.Version
		}
	case map[string]string:
		versionMap = ty
	}

	// if len(mparts) == 0, then the murmur hash value is 0
	buffer := bytes.NewBufferString("")
	keys := []string{}
	for k := range versionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(versionMap[k])
	}
	return util.GetMurmur3Hash(buffer.Bytes())
}

func UpdateDocumentState(c DatabaseClient, cpeMac string, m *common.EventMessage, fields log.Fields) error {
	// TODO: original config-version-report for ble, NO-OP for now
	if len(m.Reports) > 0 {
		return nil
	}

	updatedTime := int(time.Now().UnixNano() / 1000000)

	// set metrics labels
	metricsAgent := "default"
	if itf, ok := fields["metrics_agent"]; ok {
		metricsAgent = itf.(string)
	}
	labels, err := c.GetRootDocumentLabels(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	labels["client"] = metricsAgent

	// rootdoc-report
	// ==== update all subdocs ====
	if m.HttpStatusCode != nil {
		// all non-304 got discarded
		if *m.HttpStatusCode != http.StatusNotModified {
			return nil
		}

		// process 304
		fields["src_caller"] = common.GetCaller()
		doc, err := c.GetDocument(cpeMac, fields)
		if err != nil {
			return common.NewError(err)
		}

		newState := common.Deployed
		errorCode := 0
		errorDetails := ""
		for groupId, oldSubdoc := range doc.Items() {
			// fix the bad condition when updated_time is negative
			if oldSubdoc.NeedsUpdateForHttp304() {
				newSubdoc := common.NewSubDocument(nil, nil, &newState, &updatedTime, &errorCode, &errorDetails)
				oldState := *oldSubdoc.State()

				if err := c.SetSubDocument(cpeMac, groupId, newSubdoc, oldState, labels, fields); err != nil {
					return common.NewError(err)
				}
			}
		}
		return nil
	}

	// subdoc-report, should have some validation already
	if m.ApplicationStatus == nil || m.Namespace == nil {
		return common.NewError(fmt.Errorf("ill-formatted event"))
	}

	state := common.Failure
	errorCodePtr := m.ErrorCode
	errorDetailsPtr := m.ErrorDetails
	if *m.ApplicationStatus == "success" {
		state = common.Deployed
		errorCode := 0
		errorCodePtr = &errorCode
		errorDetails := ""
		errorDetailsPtr = &errorDetails
	} else if *m.ApplicationStatus == "pending" {
		return nil
	}

	targetGroupId := *m.Namespace
	subdoc, err := c.GetSubDocument(cpeMac, *m.Namespace)
	if err != nil {
		return common.NewError(err)
	}

	var oldState int
	if subdoc.State() != nil {
		oldState = *subdoc.State()
		if oldState < common.Deployed || oldState > common.Failure {
			err := common.Http404Error{
				Message: fmt.Sprintf("invalid state(%v) in db", oldState),
			}
			return common.NewError(err)
		}
	}

	if subdoc.UpdatedTime() != nil {
		docUpdatedTime := *subdoc.UpdatedTime()
		if docUpdatedTime < 0 {
			err := common.Http404Error{
				Message: fmt.Sprintf("invalid updated_time(%v) in db", docUpdatedTime),
			}
			return common.NewError(err)
		}
	}

	newSubdoc := common.NewSubDocument(nil, nil, &state, &updatedTime, errorCodePtr, errorDetailsPtr)

	// metricsAgent handling
	if m.MetricsAgent != nil {
		labels["client"] = *m.MetricsAgent
	}

	err = c.SetSubDocument(cpeMac, targetGroupId, newSubdoc, oldState, labels, fields)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func UpdateSubDocument(c DatabaseClient, cpeMac, subdocId string, newSubdoc, oldSubdoc *common.SubDocument, fields log.Fields) error {
	var oldState int
	if oldSubdoc != nil && oldSubdoc.State() != nil {
		oldState = *oldSubdoc.State()
	}
	// set metrics labels
	labels, err := c.GetRootDocumentLabels(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	labels["client"] = "default"

	updatedTime := int(time.Now().UnixNano() / 1000000)
	newSubdoc.SetUpdatedTime(&updatedTime)
	newState := common.InDeployment
	newSubdoc.SetState(&newState)
	err = c.SetSubDocument(cpeMac, subdocId, newSubdoc, oldState, labels, fields)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func WriteDocumentFromUpstream(c DatabaseClient, cpeMac, upstreamRespEtag string, newDoc *common.Document, d *common.Document, toDelete bool, fields log.Fields) error {
	newRootVersion := upstreamRespEtag
	if d.RootVersion() != newRootVersion {
		err := c.SetRootDocumentVersion(cpeMac, newRootVersion)
		if err != nil {
			return common.NewError(err)
		}
	}

	oldMap := map[string]struct{}{}
	for k := range d.Items() {
		oldMap[k] = struct{}{}
	}

	// need to set "state" to proper values like the download is complete
	for subdocId, newSubdoc := range newDoc.Items() {
		oldSubdoc := d.SubDocument(subdocId)
		err := UpdateSubDocument(c, cpeMac, subdocId, &newSubdoc, oldSubdoc, fields)
		if err != nil {
			return common.NewError(err)
		}
		delete(oldMap, subdocId)
	}

	if toDelete {
		for subdocId := range oldMap {
			err := c.DeleteSubDocument(cpeMac, subdocId)
			if err != nil {
				return common.NewError(err)
			}
		}
	}
	return nil
}

// this d should be a "filtered" document for download
func UpdateDocumentStateIndeployment(c DatabaseClient, cpeMac string, d *common.Document, fields log.Fields) error {
	// skip version, payload change
	newState := common.InDeployment
	updatedTime := int(time.Now().UnixNano() / 1000000)
	errorCode := 0
	errorDetails := ""

	// set metrics labels
	labels, err := c.GetRootDocumentLabels(cpeMac)
	if err != nil {
		return common.NewError(err)
	}
	labels["client"] = "default"

	for subdocId, subdoc := range d.Items() {
		if subdoc.State() != nil && (*subdoc.State() == common.Deployed || *subdoc.State() == common.InDeployment) {
			continue
		}
		newSubdoc := common.NewSubDocument(nil, nil, &newState, &updatedTime, &errorCode, &errorDetails)
		oldState := *subdoc.State()
		err := c.SetSubDocument(cpeMac, subdocId, newSubdoc, oldState, labels, fields)
		if err != nil {
			return common.NewError(err)
		}
	}
	return nil
}

func RebuildDeviceVersionMap(versions []string, cloudVersionMap map[string]string) map[string]string {
	revCloudMap := make(map[string]string)
	for k, v := range cloudVersionMap {
		revCloudMap[v] = k
	}
	m := map[string]string{
		"root": versions[0],
	}
	for _, v := range versions {
		if subdocId, ok := revCloudMap[v]; ok {
			m[subdocId] = v
		}
	}
	return m
}

func RefreshRootDocumentVersion(doc *common.Document) {
	versionMap := doc.VersionMap()
	rootVersion := HashRootVersion(versionMap)
	rootDoc := doc.GetRootDocument()
	if rootDoc != nil {
		rootDoc.Version = rootVersion
	}
}

func PreprocessRootDocument(c DatabaseClient, rHeader http.Header, mac, partnerId string, fields log.Fields) (*common.RootDocument, error) {
	fieldsDict := make(util.Dict)
	fieldsDict.Update(fields)

	// ==== deviceRootDocument should always be created from request header ====
	var bitmap int
	var err error
	supportedDocs := rHeader.Get(common.HeaderSupportedDocs)
	if len(supportedDocs) > 0 {
		bitmap, err = util.GetCpeBitmap(supportedDocs)
		if err != nil {
			log.WithFields(fields).Warn(common.NewError(err))
		}
	}

	schemaVersion := strings.ToLower(rHeader.Get(common.HeaderSchemaVersion))
	modelName := rHeader.Get(common.HeaderModelName)
	firmwareVersion := rHeader.Get(common.HeaderFirmwareVersion)

	// start with an empty rootDocument.Version, just in case there are errors in parsing the version from headers
	deviceRootDocument := common.NewRootDocument(bitmap, firmwareVersion, modelName, partnerId, schemaVersion, "", "")

	// ==== read the cloudRootDocument from db ====
	cloudRootDocument, err := c.GetRootDocument(mac)
	if err != nil {
		if !c.IsDbNotFound(err) {
			return cloudRootDocument, common.NewError(err)
		}
		cloudRootDocument = deviceRootDocument.Clone()
	} else {
		cloudRootDocument.Update(deviceRootDocument)
	}

	if err := c.SetRootDocument(mac, cloudRootDocument); err != nil {
		return cloudRootDocument, common.NewError(err)
	}
	return cloudRootDocument, nil
}
