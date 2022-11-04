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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v4"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

func TestSupplementaryApi(t *testing.T) {
	// t.Skip() // TOFIX
	log.SetOutput(ioutil.Discard)

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
	rbytes, err := ioutil.ReadAll(res.Body)
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
	assert.Equal(t, profile1["name"].(string), "xpc_test_profile_001")

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
    "Description":"XPC TEST PROFILE 001",
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
      "name": "xpc_test_profile_001",
      "versionHash": "55e295e3",
      "value": {
        "Description": "XPC TEST PROFILE 001",
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
	t.Skip("SKIP telemetry testing for now")
	log.SetOutput(ioutil.Discard)

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
