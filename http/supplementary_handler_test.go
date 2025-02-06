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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v4"
	"gotest.tools/assert"
)

func TestSupplementaryApi(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	//assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	output := common.TR181Output{}
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes := []byte(output.Parameters[0].Value)

	var itf util.Dict
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok := itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok := profilesItf.([]interface{})
	assert.Assert(t, ok)

	profile1Itf := profilesJs[0]

	profile1, ok := profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok := profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok := coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	var srcItf map[string]interface{}
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)
}

const (
	rawProfileStr = `
{
    "Description":"X TEST PROFILE 001",
    "Version":"0.1",
    "Protocol":"HTTP",
    "EncodingType":"JSON",
    "ReportingInterval":60,
    "TimeReference":"0001-01-01T00:00:00Z",
    "ActivationTimeout":600,
    "Parameter": [
        {"type":"dataModel", "reference":"Profile.Name"} ,
        {"type":"dataModel", "reference":"Profile.Description"} ,
        {"type":"dataModel", "reference":"Profile.Version"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.MaxBitRate"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.OperatingFrequencyBand"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.ChannelsInUse"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Channel"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.AutoChannelEnable"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.OperatingChannelBandwidth"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.RadioResetCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.PacketsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.PacketsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.ErrorsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.ErrorsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.DiscardPacketsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.DiscardPacketsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.PLCPErrorCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.FCSErrorCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_NoiseFloor"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.Noise"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ChannelUtilization"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ActivityFactor"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.MaxBitRate"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.OperatingFrequencyBand"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.ChannelsInUse"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Channel"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.AutoChannelEnable"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.OperatingChannelBandwidth"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.RadioResetCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.PacketsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.PacketsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.ErrorsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.ErrorsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.DiscardPacketsSent"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.DiscardPacketsReceived"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.PLCPErrorCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.FCSErrorCount"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_NoiseFloor"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.Noise"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ChannelUtilization"} ,
        {"type":"dataModel", "reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ActivityFactor"}
    ],
    "HTTP": {
        "URL":"https://rdkrtldev.stb.r53.xcal.tv/",
        "Compression":"None",
        "Method":"POST",
        "RequestURIParameter": [
            {"Name":"profileName", "Reference":"Profile.Name" },
            {"Name":"reportVersion", "Reference":"Profile.Version" }
        ]
 
    },
    "JSONEncoding": {
        "ReportFormat":"NameValuePair",
        "ReportTimestamp": "None"
    }
}
`

	mockProfileResponse = `
{
  "profiles": [
    {
      "name": "x_test_profile_001",
      "versionHash": "55e295e3",
      "value": {
        "Description": "X TEST PROFILE 001",
        "Version": "0.1",
        "Protocol": "HTTP",
        "EncodingType": "JSON",
        "ReportingInterval": 60,
        "TimeReference": "0001-01-01T00:00:00Z",
        "ActivationTimeout": 600,
        "Parameter": [
          {
            "type": "dataModel",
            "reference": "Profile.Name"
          },
          {
            "type": "dataModel",
            "reference": "Profile.Description"
          },
          {
            "type": "dataModel",
            "reference": "Profile.Version"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.MaxBitRate"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.OperatingFrequencyBand"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.ChannelsInUse"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Channel"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.AutoChannelEnable"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.OperatingChannelBandwidth"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.RadioResetCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.PacketsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.PacketsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.ErrorsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.ErrorsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.DiscardPacketsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.DiscardPacketsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.PLCPErrorCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.FCSErrorCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.X_COMCAST-COM_NoiseFloor"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.Noise"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ChannelUtilization"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ActivityFactor"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.MaxBitRate"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.OperatingFrequencyBand"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.ChannelsInUse"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Channel"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.AutoChannelEnable"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.OperatingChannelBandwidth"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.RadioResetCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.PacketsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.PacketsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.ErrorsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.ErrorsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.DiscardPacketsSent"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.DiscardPacketsReceived"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.PLCPErrorCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.FCSErrorCount"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.X_COMCAST-COM_NoiseFloor"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.Noise"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ChannelUtilization"
          },
          {
            "type": "dataModel",
            "reference": "Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ActivityFactor"
          }
        ],
        "HTTP": {
          "URL": "https://rdkrtldev.stb.r53.xcal.tv/",
          "Compression": "None",
          "Method": "POST",
          "RequestURIParameter": [
            {
              "Name": "profileName",
              "Reference": "Profile.Name"
            },
            {
              "Name": "reportVersion",
              "Reference": "Profile.Version"
            }
          ]
        },
        "JSONEncoding": {
          "ReportFormat": "NameValuePair",
          "ReportTimestamp": "None"
        }
      }
    }
  ]
}`
)

const (
	mockProfileNotFoundResponse = `"<h2>404 NOT FOUND</h2>profiles not found"`
)

func TestSupplementaryApiNoDataInXconf(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(mockProfileNotFoundResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682X")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusNotFound)
}

func TestSupplementaryWithExtraQueryParams(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(common.HeaderReqUrl, r.URL.String())
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 set up the query params ====
	bitmap1 := 32479
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	etag := strconv.Itoa(int(time.Now().Unix()))
	queryParams1 := "stormReadyWifi=true"
	srcDoc1 := common.NewRootDocument(bitmap1, firmwareVersion1, modelName1, partner1, schemaVersion1, etag, queryParams1)
	bbytes, err := json.Marshal(srcDoc1)
	assert.NilError(t, err)

	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err := http.NewRequest("POST", rootdocUrl, bytes.NewReader(bbytes))
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 2 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// TODO verify the resHeader
	xReqUrl := res.Header.Get(common.HeaderReqUrl)
	assert.Assert(t, strings.Contains(xReqUrl, queryParams1))

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	output := common.TR181Output{}
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes := []byte(output.Parameters[0].Value)

	var itf util.Dict
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok := itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok := profilesItf.([]interface{})
	assert.Assert(t, ok)

	profile1Itf := profilesJs[0]

	profile1, ok := profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok := profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok := coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	var srcItf map[string]interface{}
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)
}

func TestSupplementaryApiBadPartnerHeader(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	validPartners := []string{"comcast", "comcast-dev", "cox", "rogers", "shaw", "sky-de", "sky-italia", "sky-italia-dev", "sky-roi", "sky-roi-dev", "sky-uk", "sky-uk-dev", "videotron"}
	server.SetValidPartners(validPartners)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "R NOT")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	//assert.Equal(t, res.StatusCode, http.StatusOK)
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	output := common.TR181Output{}
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes := []byte(output.Parameters[0].Value)

	var itf util.Dict
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok := itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok := profilesItf.([]interface{})
	assert.Assert(t, ok)

	profile1Itf := profilesJs[0]

	profile1, ok := profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok := profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok := coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	var srcItf map[string]interface{}
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)
}

func TestSupplementaryAppendingFlag(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock server ====
	var ss string
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ss = r.URL.String()
			w.Header().Set(common.HeaderReqUrl, r.URL.String())
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 set up the query params ====
	bitmap1 := 32479
	firmwareVersion1 := "CGM4331COM_4.11p7s1_PROD_sey"
	modelName1 := "CGM4331COM"
	partner1 := "comcast"
	schemaVersion1 := "33554433-1.3,33554434-1.3"
	etag := strconv.Itoa(int(time.Now().Unix()))
	queryParams1 := "stormReadyWifi=true"
	srcDoc1 := common.NewRootDocument(bitmap1, firmwareVersion1, modelName1, partner1, schemaVersion1, etag, queryParams1)
	bbytes, err := json.Marshal(srcDoc1)
	assert.NilError(t, err)

	rootdocUrl := fmt.Sprintf("/api/v1/device/%v/rootdocument", cpeMac)
	req, err := http.NewRequest("POST", rootdocUrl, bytes.NewReader(bbytes))
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 2 set append flag true ====
	appendEnabled := true
	server.SetSupplementaryAppendingEnabled(appendEnabled)
	assert.Assert(t, appendEnabled == server.SupplementaryAppendingEnabled())

	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err = http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// TODO verify the resHeader
	xReqUrl := res.Header.Get(common.HeaderReqUrl)
	assert.Assert(t, strings.Contains(xReqUrl, queryParams1))

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	output := common.TR181Output{}
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes := []byte(output.Parameters[0].Value)

	var itf util.Dict
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok := itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok := profilesItf.([]interface{})
	assert.Assert(t, ok)

	profile1Itf := profilesJs[0]

	profile1, ok := profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok := profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok := coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	var srcItf map[string]interface{}
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)

	// ==== step 3 verify the query params ====
	assert.Assert(t, strings.Contains(ss, "&stormReadyWifi=true"))

	// ==== step 4 set append flag false ====
	appendEnabled = false
	server.SetSupplementaryAppendingEnabled(appendEnabled)
	assert.Assert(t, appendEnabled == server.SupplementaryAppendingEnabled())

	req, err = http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res = ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	// ==== step 5 verify the query params ====
	assert.Assert(t, !strings.Contains(ss, "&stormReadyWifi=true"))
}

func TestSupplementaryApiBadRequest(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)
	validPartners := []string{"comcast", "comcast-dev", "cox", "rogers", "shaw", "sky-de", "sky-italia", "sky-italia-dev", "sky-roi", "sky-roi-dev", "sky-uk", "sky-uk-dev", "videotron"}
	server.SetValidPartners(validPartners)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request"))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "R NOT")
	req.Header.Set(common.HeaderAccountID, "ERROR: ld.so: object '/usr/lib/libwayland-egl.so.0' from LD_PRE")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)
}

func TestSupplementaryXconfTimeout(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock server ====
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(1) * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockProfileResponse))
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)
	server.XconfConnector.SetXconfHost(mockServer.URL)
	server.XconfConnector.HttpClient.Client.Timeout = time.Duration(500) * time.Millisecond

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusGatewayTimeout)
}

type errorRoundTripper struct{}

func (ert *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Return a response with an error-inducing reader
	return &http.Response{
		Body: io.NopCloser(&errorReader{}),
	}, nil
}

// errorReader simulates an error when reading
type errorReader struct{}

func (er *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("context deadline exceeded (Client.Timeout or context cancellation while reading body)")
}

func TestSupplementaryXconfReadAllErr(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== setup mock client ====
	mockedClient := &http.Client{
		Transport: &errorRoundTripper{},
	}

	server.XconfConnector.HttpClient.Client = mockedClient

	// ==== setup data ====
	cpeMac := util.GenerateRandomCpeMac()

	// ==== step 1 verify /config expect 200 with 1 mpart ====
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)

	// add headers
	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, "TG1682G")
	req.Header.Set(common.HeaderPartnerID, "comcast")
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")

	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	_, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusGatewayTimeout)
}

var (
	mockUpstreamProfileResponse1 = []byte(`[
  {
    "name": "subname1",
    "value": {
      "Parameter": [
        {
          "reference": "Device.X_RDK_GatewayManagement.Gateway.1.MacAddress",
          "reportEmpty": true,
          "type": "dataModel"
        }
      ]
    },
    "versionHash": "977e16c4"
  },
  {
    "name": "subname2",
    "value": {
      "Parameter": [
        {
          "reference": "Device.X_RDK_GatewayManagement.Gateway.2.MacAddress",
          "reportEmpty": true,
          "type": "dataModel"
        }
      ]
    },
    "versionHash": "4f207ebd"
  }
]`)
)

func TestSupplementaryUpstreamProfiles(t *testing.T) {
	log.SetOutput(io.Discard)

	server := NewWebconfigServer(sc, true)
	router := server.GetRouter(true)

	// ==== step 1 setup mock xconf server ====
	cxbytes, err := util.CompactJson([]byte(mockProfileResponse))
	assert.NilError(t, err)
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(cxbytes)
		}))
	defer mockServer.Close()
	server.XconfConnector.SetXconfHost(mockServer.URL)

	// ==== step 2 set up upstream mock server ====
	cubytes, err := util.CompactJson(mockUpstreamProfileResponse1)
	assert.NilError(t, err)
	mockUpstreamServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(cubytes)
		}))
	defer mockUpstreamServer.Close()
	server.SetUpstreamHost(mockUpstreamServer.URL)
	targetUpstreamHost := server.UpstreamHost()
	assert.Equal(t, mockUpstreamServer.URL, targetUpstreamHost)

	// ==== step 3 verify /config expect 200 with 1 mpart ====
	cpeMac := util.GenerateRandomCpeMac()
	configUrl := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", configUrl, nil)
	assert.NilError(t, err)

	// add headers
	modelName := "TG1682G"
	partnerID := "comcast"
	firmwareVersion := "TG1682_3.14p9s6_PROD_sey"

	req.Header.Set(common.HeaderSupplementaryService, "telemetry")
	req.Header.Set(common.HeaderProfileVersion, "2.0")
	req.Header.Set(common.HeaderModelName, modelName)
	req.Header.Set(common.HeaderPartnerID, partnerID)
	req.Header.Set(common.HeaderAccountID, "1234567890")
	req.Header.Set(common.HeaderFirmwareVersion, firmwareVersion)

	res := ExecuteRequest(req, router).Result()
	rbytes, err := io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err := util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok := mparts["telemetry"]
	assert.Assert(t, ok)

	output := common.TR181Output{}
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes := []byte(output.Parameters[0].Value)

	var itf util.Dict
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok := itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok := profilesItf.([]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(profilesJs) == 1)
	profile1Itf := profilesJs[0]

	profile1, ok := profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok := profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok := coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	var srcItf map[string]interface{}
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)

	// ==== step 4 enable the feature flag but no query_param expect 200 with 1 mpart ====
	server.SetUpstreamProfilesEnabled(true)
	defer server.SetUpstreamProfilesEnabled(false)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok = mparts["telemetry"]
	assert.Assert(t, ok)

	output = *new(common.TR181Output)
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes = []byte(output.Parameters[0].Value)

	itf = make(util.Dict)
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok = itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok = profilesItf.([]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(profilesJs) == 1)
	profile1Itf = profilesJs[0]
	profile1, ok = profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	coreProfile1Itf, ok = profile1["value"]
	assert.Assert(t, ok)
	coreProfile1, ok = coreProfile1Itf.(map[string]interface{})
	assert.Assert(t, ok)

	srcItf = make(map[string]interface{})
	err = json.Unmarshal([]byte(rawProfileStr), &srcItf)
	assert.NilError(t, err)
	assert.DeepEqual(t, coreProfile1, srcItf)

	// ==== step 5 set query_param in the root_document expect 200 with 1 mpart ====
	rdoc := new(common.RootDocument)
	rdoc.QueryParams = "key1=val1"
	rdoc.ModelName = modelName
	rdoc.PartnerId = partnerID
	rdoc.FirmwareVersion = firmwareVersion
	err = server.SetRootDocument(cpeMac, rdoc)
	assert.NilError(t, err)

	res = ExecuteRequest(req, router).Result()
	rbytes, err = io.ReadAll(res.Body)
	assert.NilError(t, err)
	res.Body.Close()
	t.Logf("%s\n", rbytes)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	mparts, err = util.ParseMultipart(res.Header, rbytes)
	assert.NilError(t, err)
	assert.Equal(t, len(mparts), 1)
	mpart, ok = mparts["telemetry"]
	assert.Assert(t, ok)

	output = *new(common.TR181Output)
	err = msgpack.Unmarshal(mpart.Bytes, &output)
	assert.NilError(t, err)
	assert.Equal(t, len(output.Parameters), 1)
	assert.Equal(t, output.Parameters[0].Name, common.TR181NameTelemetry)
	assert.Equal(t, output.Parameters[0].DataType, common.TR181Blob)
	mbytes = []byte(output.Parameters[0].Value)

	itf = make(util.Dict)
	err = msgpack.Unmarshal(mbytes, &itf)
	assert.NilError(t, err)

	_, err = json.Marshal(&itf)
	assert.NilError(t, err)

	// assume only 1 "profile" is returned
	profilesItf, ok = itf["profiles"]
	assert.Assert(t, ok)
	profilesJs, ok = profilesItf.([]interface{})
	assert.Assert(t, ok)
	assert.Assert(t, len(profilesJs) == 3)
	profile1Itf = profilesJs[0]
	profile1, ok = profile1Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile1["name"].(string), "x_test_profile_001")

	profile2Itf := profilesJs[1]
	profile2, ok := profile2Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile2["name"].(string), "subname1")

	profile3Itf := profilesJs[2]
	profile3, ok := profile3Itf.(map[string]interface{})
	assert.Assert(t, ok)
	assert.Equal(t, profile3["name"].(string), "subname2")
}
