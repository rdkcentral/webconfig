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
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
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
	// if len(mac) != 12 {
	// 	err := common.NewError(fmt.Errorf("Ill-formatted mac %v", mac))
	// 	return nil, nil, deviceRootDocument, nil, false, common.NewError(err)
	// }

	// get version map
	deviceVersionMap, err := parseVersionMap(rHeader, fieldsDict)
	if err != nil {
		return nil, nil, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
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

	switch rootCmpEnum {
	case common.RootDocumentEquals:
		// create an empty "document"
		document := common.NewDocument(cloudRootDocument)
		// no need to update root doc
		return document, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, nil
	case common.RootDocumentVersionOnlyChanged, common.RootDocumentMissing:
		// meta unchanged but subdoc versions change ==> new configs
		// getDoc, then filter
		document, err := c.GetDocument(mac)
		if err != nil {
			// 404 should be included here
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
		document.SetRootDocument(cloudRootDocument)
		filteredDocument := document.FilterForGet(deviceVersionMap)
		for _, subdocId := range c.BlockedSubdocIds() {
			filteredDocument.DeleteSubDocument(subdocId)
		}
		return filteredDocument, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, nil
	case common.RootDocumentMetaChanged:
		// getDoc, send it upstream
		document, err := c.GetDocument(mac)
		if err != nil {
			// 404 should be included here
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
		document.SetRootDocument(cloudRootDocument)

		// need to update rootDoc meta
		// NOTE need to clone the deviceRootDocument and set the version "" to avoid device root update was set back to cloud
		clonedRootDoc := deviceRootDocument.Clone()
		clonedRootDoc.Version = ""
		if err := c.SetRootDocument(mac, clonedRootDoc); err != nil {
			return nil, cloudRootDocument, deviceRootDocument, deviceVersionMap, false, common.NewError(err)
		}
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
func parseVersionMap(rHeader http.Header, fieldsDict util.Dict) (map[string]string, error) {
	deviceVersionMap := make(map[string]string)

	queryStr := rHeader.Get(common.HeaderDocName)
	subdocIds := strings.Split(queryStr, ",")
	if len(queryStr) == 0 {
		return deviceVersionMap, nil
	}

	ifNoneMatch := rHeader.Get(common.HeaderIfNoneMatch)
	versions := strings.Split(ifNoneMatch, ",")

	if len(subdocIds) != len(versions) {
		err := fmt.Errorf("group_id=%v  IF-NONE-MATCH=%v", queryStr, ifNoneMatch)
		return nil, common.NewError(err)
	}

	for i, subdocId := range subdocIds {
		deviceVersionMap[subdocId] = versions[i]
	}
	return deviceVersionMap, nil
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
	for _, version := range versionMap {
		buffer.WriteString(version)
	}
	return util.GetMurmur3Hash(buffer.Bytes())
}

func UpdateDocumentState(c DatabaseClient, cpeMac string, m *common.EventMessage, fields log.Fields) error {
	// TODO: original config-version-report for ble, NO-OP for now
	if len(m.Reports) > 0 {
		return nil
	}

	updatedTime := int(time.Now().UnixNano() / 1000000)

	// rootdoc-report
	// ==== update all subdocs ====
	if m.HttpStatusCode != nil {
		// all non-304 got discarded
		if *m.HttpStatusCode != http.StatusNotModified {
			return nil
		}

		// process 304
		doc, err := c.GetDocument(cpeMac)
		if err != nil {
			return common.NewError(err)
		}

		newState := common.Deployed
		errorCode := 0
		errorDetails := ""
		for groupId, oldSubdoc := range doc.Items() {
			// fix the bad condition when updated_time is negative
			if oldSubdoc.State() != nil && *oldSubdoc.State() != common.Deployed || oldSubdoc.UpdatedTime() != nil && *oldSubdoc.UpdatedTime() < 0 {
				newSubdoc := common.NewSubDocument(nil, nil, &newState, &updatedTime, &errorCode, &errorDetails)
				oldState := *oldSubdoc.State()

				var metricsAgent string
				if itf, ok := fields["metrics_agent"]; ok {
					metricsAgent = itf.(string)
				}
				if err := c.SetSubDocument(cpeMac, groupId, newSubdoc, oldState, metricsAgent); err != nil {
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
	if *m.ApplicationStatus == "success" {
		state = common.Deployed
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

	newSubdoc := common.NewSubDocument(nil, nil, &state, &updatedTime, m.ErrorCode, m.ErrorDetails)

	// metricsAgent handling
	var metricsAgent string
	if m.MetricsAgent != nil {
		metricsAgent = *m.MetricsAgent
	}

	err = c.SetSubDocument(cpeMac, targetGroupId, newSubdoc, oldState, metricsAgent)
	if err != nil {
		return common.NewError(err)
	}
	return nil
}

func UpdateSubDocument(c DatabaseClient, cpeMac string, mpart *common.Multipart, subdoc *common.SubDocument) error {
	changed := false
	if subdoc == nil {
		changed = true
	} else {
		if *subdoc.Version() != mpart.Version {
			changed = true
		}
	}
	if changed {
		newState := common.InDeployment
		updatedTime := int(time.Now().UnixNano() / 1000000)
		errorCode := 0
		errorDetails := ""
		newSubdoc := common.NewSubDocument(mpart.Bytes, &mpart.Version, &newState, &updatedTime, &errorCode, &errorDetails)
		oldState := *subdoc.State()
		metricsAgent := ""
		err := c.SetSubDocument(cpeMac, mpart.Name, newSubdoc, oldState, metricsAgent)
		if err != nil {
			return common.NewError(err)
		}
		// SetSubDocument(string, string, *common.SubDocument, ...interface{}) error
		// c.SetSubDocument(cpeMac, groupId, newSubdoc, oldState, metricsAgent); err != nil {
	}

	return nil
}

func WriteDocumentFromUpstream(c DatabaseClient, cpeMac, upstreamRespEtag string, mparts []common.Multipart, d *common.Document) error {
	newRootVersion := upstreamRespEtag
	if d.RootVersion() == newRootVersion {
		return nil
	}

	err := c.SetRootDocumentVersion(cpeMac, newRootVersion)
	if err != nil {
		return common.NewError(err)
	}

	// need to set "state" to proper values like the download is complete
	for _, mpart := range mparts {
		subdoc := d.SubDocument(mpart.Name)
		err := UpdateSubDocument(c, cpeMac, &mpart, subdoc)
		if err != nil {
			return common.NewError(err)
		}
	}
	return nil
}

// this d should be a "filtered" document for download
func UpdateDocumentStateIndeployment(c DatabaseClient, cpeMac string, d *common.Document) error {
	// skip version, payload change
	newState := common.InDeployment
	metricsAgent := ""
	updatedTime := int(time.Now().UnixNano() / 1000000)
	errorCode := 0
	errorDetails := ""

	for subdocId, subdoc := range d.Items() {
		if subdoc.State() != nil && (*subdoc.State() == common.Deployed || *subdoc.State() == common.InDeployment) {
			continue
		}
		newSubdoc := common.NewSubDocument(nil, nil, &newState, &updatedTime, &errorCode, &errorDetails)
		oldState := *subdoc.State()
		err := c.SetSubDocument(cpeMac, subdocId, newSubdoc, oldState, metricsAgent)
		if err != nil {
			return common.NewError(err)
		}
	}
	return nil
}
