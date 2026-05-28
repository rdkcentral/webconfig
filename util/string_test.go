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
package util

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rdkcentral/webconfig/common"
	"gotest.tools/assert"
)

func TestString(t *testing.T) {
	s := "112233445566"
	c := ToColonMac(s)
	expected := "11:22:33:44:55:66"
	assert.Equal(t, c, expected)
}

func TestGetAuditId(t *testing.T) {
	auditId := GetAuditId()
	assert.Equal(t, len(auditId), 32)
}

func TestTelemetryQuery(t *testing.T) {
	header := http.Header{}
	header.Set(common.HeaderProfileVersion, "2.0")
	header.Set(common.HeaderModelName, "TG1682G")
	header.Set(common.HeaderPartnerID, "comcast")
	header.Set(common.HeaderAccountID, "1234567890")
	header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")
	mac := "567890ABCDEF"
	qstr := GetTelemetryQueryString(header, mac, "", "comcast")

	expected := "env=PROD&partnerId=comcast&version=2.0&model=TG1682G&accountId=1234567890&firmwareVersion=TG1682_3.14p9s6_PROD_sey&estbMacAddress=567890ABCDF1&ecmMacAddress=567890ABCDEF"
	assert.Equal(t, qstr, expected)

	// with queryParams
	queryParams := "stormReadyWifi=true"
	qstr = GetTelemetryQueryString(header, mac, queryParams, "comcast")
	expected = "env=PROD&partnerId=comcast&version=2.0&model=TG1682G&accountId=1234567890&firmwareVersion=TG1682_3.14p9s6_PROD_sey&estbMacAddress=567890ABCDF1&ecmMacAddress=567890ABCDEF&stormReadyWifi=true"
	assert.Equal(t, qstr, expected)
}

func TestIsValidUTF8(t *testing.T) {
	b1 := []byte(`{"foo":"bar","hello":123,"world":true}`)
	assert.Assert(t, IsValidUTF8(b1))

	b2 := common.RandomBytes(100, 150)
	assert.Assert(t, !IsValidUTF8(b2))
}

func TestTelemetryQueryWithWanMac(t *testing.T) {
	header := http.Header{}
	header.Set(common.HeaderProfileVersion, "2.0")
	header.Set(common.HeaderModelName, "TG1682G")
	header.Set(common.HeaderPartnerID, "comcast")
	header.Set(common.HeaderAccountID, "1234567890")
	header.Set(common.HeaderFirmwareVersion, "TG1682_3.14p9s6_PROD_sey")
	mac := "567890ABCDEF"
	header.Set(common.HeaderWanMac, "567890ABCDEF")
	qstr := GetTelemetryQueryString(header, mac, "", "comcast")

	expected := "env=PROD&partnerId=comcast&version=2.0&model=TG1682G&accountId=1234567890&firmwareVersion=TG1682_3.14p9s6_PROD_sey&estbMacAddress=567890ABCDEF"
	assert.Equal(t, qstr, expected)
}

func TestAppendProfiles(t *testing.T) {
	mockedBaseProfilesResponse := `{"profiles":[{"name":"XfinityWIFI_SYNC","value":{"Description":"XfinityWIFI_SYNC to capture XWIFI info every 12 hours","EncodingType":"JSON","HTTP":{"Compression":"None","Method":"POST","RequestURIParameter":[{"Name":"profileName","Reference":"Profile.Name"},{"Name":"reportVersion","Reference":"Profile.Version"}],"URL":"https://stbrtl.stb.r53.xcal.tv"},"JSONEncoding":{"ReportFormat":"NameValuePair","ReportTimestamp":"None"},"Parameter":[{"reference":"Profile.Name","type":"dataModel"},{"reference":"Profile.Version","type":"dataModel"},{"name":"Profile","reference":"Device.DeviceInfo.X_RDK_RDKProfileName","type":"dataModel"},{"name":"Time","reference":"Device.Time.X_RDK_CurrentUTCTime","type":"dataModel"},{"name":"mac","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_MAC","type":"dataModel"},{"name":"CMMAC_split","reference":"Device.DeviceInfo.X_COMCAST-COM_CM_MAC","type":"dataModel"},{"name":"erouterIpv4","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_IP","type":"dataModel"},{"name":"erouterIpv6","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_IPv6","type":"dataModel"},{"name":"PartnerId","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_Syndication.PartnerId","type":"dataModel"},{"name":"Version","reference":"Device.DeviceInfo.SoftwareVersion","type":"dataModel"},{"name":"AccountId","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_RFC.Feature.AccountInfo.AccountID","type":"dataModel"},{"name":"cpe_passpoint_enable","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_InterworkingServiceEnable","type":"dataModel"},{"name":"cpe_passpoint_inter_parameters","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_InterworkingService.Parameters","type":"dataModel"},{"name":"cpe_passpoint_parameters","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_Passpoint.Parameters","type":"dataModel"},{"name":"cpe_passpoint_rdk_enable","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_Passpoint.Enable","type":"dataModel"},{"name":"open5_bss_active","reference":"Device.WiFi.SSID.6.Enable","type":"dataModel"},{"name":"secure5_bss_active","reference":"Device.WiFi.SSID.10.Enable","type":"dataModel"},{"name":"secure5_radius_server_ip","reference":"Device.WiFi.AccessPoint.10.Security.RadiusServerIPAddr","type":"dataModel"},{"name":"wifi_enabled","reference":"Device.DeviceInfo.X_COMCAST_COM_xfinitywifiEnable","type":"dataModel"},{"name":"primary_tunnel","reference":"Device.X_COMCAST-COM_GRE.Tunnel.1.PrimaryRemoteEndpoint","type":"dataModel"},{"name":"secondary_tunnel","reference":"Device.X_COMCAST-COM_GRE.Tunnel.1.SecondaryRemoteEndpoint","type":"dataModel"},{"name":"open5_advertisement_str","reference":"Device.WiFi.AccessPoint.6.SSIDAdvertisementEnabled","type":"dataModel"},{"name":"secure5_advertisement_str","reference":"Device.WiFi.AccessPoint.10.SSIDAdvertisementEnabled","type":"dataModel"},{"name":"device_vlan_2","reference":"Device.X_COMCAST-COM_GRE.Tunnel.1.Interface.2.VLANID","type":"dataModel"},{"name":"open5_ssid_str","reference":"Device.WiFi.SSID.6.SSID","type":"dataModel"},{"name":"secure5_ssid_str","reference":"Device.WiFi.SSID.10.SSID","type":"dataModel"},{"name":"secure5_securitymode_str","reference":"Device.WiFi.AccessPoint.10.Security.ModeEnabled","type":"dataModel"},{"name":"secure5_pri_port","reference":"Device.WiFi.AccessPoint.10.Security.RadiusServerPort","type":"dataModel"},{"name":"dscp_marker","reference":"Device.X_COMCAST-COM_GRE.Tunnel.1.DSCPMarkPolicy","type":"dataModel"},{"name":"WIFI_CH_2_split","reference":"Device.WiFi.Radio.2.Channel","type":"dataModel"},{"name":"WIFI_CW_2_split","reference":"Device.WiFi.Radio.2.OperatingChannelBandwidth","type":"dataModel"},{"name":"open5_bssid_str","reference":"Device.WiFi.SSID.6.BSSID","type":"dataModel"},{"name":"secure5_bssid_str","reference":"Device.WiFi.SSID.10.BSSID","type":"dataModel"},{"name":"open5_status_str","reference":"Device.WiFi.SSID.6.Status","type":"dataModel"},{"name":"secure5_status_str","reference":"Device.WiFi.SSID.10.Status","type":"dataModel"},{"name":"open5_beaconpower_str","reference":"Device.WiFi.AccessPoint.6.X_RDKCENTRAL-COM_ManagementFramePowerControl","type":"dataModel"},{"name":"open5_beaconrate_str","reference":"Device.WiFi.AccessPoint.6.X_RDKCENTRAL-COM_BeaconRate","type":"dataModel"},{"name":"secure5_beaconpower_str","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_ManagementFramePowerControl","type":"dataModel"},{"name":"secure5_beaconrate_str","reference":"Device.WiFi.AccessPoint.10.X_RDKCENTRAL-COM_BeaconRate","type":"dataModel"},{"name":"radio5_beaconinterval","reference":"Device.WiFi.Radio.2.X_COMCAST-COM_BeaconInterval","type":"dataModel"},{"name":"secure5_encryption","reference":"Device.WiFi.AccessPoint.10.Security.X_CISCO_COM_EncryptionMethod","type":"dataModel"},{"name":"secure5_sec_radius_server_ip","reference":"Device.WiFi.AccessPoint.10.Security.SecondaryRadiusServerIPAddr","type":"dataModel"},{"name":"secure5_sec_port","reference":"Device.WiFi.AccessPoint.10.Security.SecondaryRadiusServerPort","type":"dataModel"},{"name":"UPTIME_split","reference":"Device.DeviceInfo.UpTime","type":"dataModel"},{"name":"open5_radius_server_ip","reference":"Device.WiFi.AccessPoint.6.Security.RadiusServerIPAddr","type":"dataModel"},{"name":"open5_pri_port","reference":"Device.WiFi.AccessPoint.6.Security.RadiusServerPort","type":"dataModel"},{"name":"open5_isolation_enable","reference":"Device.WiFi.AccessPoint.6.IsolationEnable","type":"dataModel"},{"name":"secure5_isolation_enable","reference":"Device.WiFi.AccessPoint.10.IsolationEnable","type":"dataModel"},{"name":"secure24_bss_active","reference":"Device.WiFi.SSID.9.Enable","type":"dataModel"},{"name":"open24_bss_active","reference":"Device.WiFi.SSID.5.Enable","type":"dataModel"}],"Protocol":"HTTP","ReportingAdjustments":{"FirstReportingInterval":300},"ReportingInterval":86400,"TimeReference":"0001-01-01T00:00:00Z","Version":"0.6"},"versionHash":"d9c6f386"},{"name":"WIFI_MOTION_Telemetry","value":{"Description":"CSCWFM_Telemetry","EncodingType":"JSON","HTTP":{"Compression":"None","Method":"POST","RequestURIParameter":[{"Name":"profileName","Reference":"Profile.Name"}],"URL":"https://stbrtl-oi.stb.r53.xcal.tv"},"JSONEncoding":{"ReportFormat":"NameValuePair","ReportTimestamp":"None"},"Parameter":[{"name":"Profile_Name","reference":"Profile.Name","type":"dataModel"},{"name":"Profile","reference":"Device.DeviceInfo.X_RDK_RDKProfileName","type":"dataModel"},{"name":"Time","reference":"Device.Time.X_RDK_CurrentUTCTime","type":"dataModel"},{"name":"mac","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_MAC","type":"dataModel"},{"name":"CMMAC_split","reference":"Device.DeviceInfo.X_COMCAST-COM_CM_MAC","type":"dataModel"},{"name":"erouterIpv4","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_IP","type":"dataModel"},{"name":"erouterIpv6","reference":"Device.DeviceInfo.X_COMCAST-COM_WAN_IPv6","type":"dataModel"},{"name":"PartnerId","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_Syndication.PartnerId","type":"dataModel"},{"name":"Version","reference":"Device.DeviceInfo.SoftwareVersion","type":"dataModel"},{"name":"AccountId","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_RFC.Feature.AccountInfo.AccountID","type":"dataModel"},{"component":"CSCWFMRXM","eventName":"CSCWFM_RXMrbussub_fail","name":"SYS_ERROR_WFMrbussub_fail","type":"event","use":"count"},{"component":"CSCWFMBRG","eventName":"CSCWFM_CSIpipe_restart","name":"SYS_SH_WFMpipe_restart","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSCWFM_ctrlifcreate_fail","name":"SYS_ERROR_WFMifcreate_fail","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSC_RXMrbusinit_fail","name":"SYS_ERROR_WFM_rbusinit_fail","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSCWFM_CSIsessionacquire_fail","name":"SYS_ERROR_WFMSessionAcq_fail","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSCWFM_RXMeventscreate_fail","name":"SYS_ERROR_WFMeventscreate_fail","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSCWFM_CSIsessionenable_fail","name":"SYS_ERROR_WFMsessionenable_fail","type":"event","use":"count"},{"component":"CSCWFMRXM","eventName":"CSCWFM_RXM_restart","name":"SYS_SH_WFMRXM_restart","type":"event","use":"count"},{"component":"CSCWFM","eventName":"CSCWFM_borg_restart","name":"SYS_SH_WFMborg_restart","type":"event","use":"count"},{"component":"CSCWFM","eventName":"CSCWFM_mqtt_restart","name":"SYS_SH_WFMmqtt_restart","type":"event","use":"count"},{"component":"CSCWFMBRG","eventName":"CSCWFM_sounding_state","name":"WFMsoundingstate_split","type":"event","use":"absolute"},{"name":"WFMEnable_split","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_RFC.Feature.CognitiveMotionDetection.Enable","type":"dataModel"},{"name":"WFMStatus_split","reference":"Device.DeviceInfo.X_RDKCENTRAL-COM_XHFW.WiFiMotionStatus","type":"dataModel"},{"logFile":"ZilkerLog.txt","marker":"wfmCogAgentCommFailCnt_split","search":"wfmCogAgentCommFailCnt:","type":"grep","use":"count"},{"logFile":"ZilkerLog.txt","marker":"wfmCogAgentConnected_split","search":"wfmCogAgentConnected:","type":"grep","use":"absolute"}],"Protocol":"HTTP","ReportingInterval":900,"TimeReference":"0001-01-01T00:00:00Z","Version":"0.1"},"versionHash":"bf86fd16"}]}`
	mockedExtraProfilesResponse := `[{"name":"james_test_profile_001","value":{"ActivationTimeout":600,"Description":"Telemetry 2.0 HSD Gateway WiFi Radio","EncodingType":"JSON","HTTP":{"Compression":"None","Method":"POST","RequestURIParameter":[{"Name":"profileName","Reference":"Profile.Name"},{"Name":"reportVersion","Reference":"Profile.Version"}],"URL":"https://rdkrtldev.stb.r53.xcal.tv/"},"JSONEncoding":{"ReportFormat":"NameValuePair","ReportTimestamp":"None"},"Parameter":[{"reference":"Profile.Name","type":"dataModel"},{"reference":"Profile.Description","type":"dataModel"},{"reference":"Profile.Version","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.MaxBitRate","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.OperatingFrequencyBand","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.ChannelsInUse","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Channel","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.AutoChannelEnable","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.OperatingChannelBandwidth","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.RadioResetCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.PacketsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.PacketsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.ErrorsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.ErrorsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.DiscardPacketsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.DiscardPacketsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.PLCPErrorCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.FCSErrorCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_NoiseFloor","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.Noise","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ChannelUtilization","type":"dataModel"},{"reference":"Device.WiFi.Radio.1.Stats.X_COMCAST-COM_ActivityFactor","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.MaxBitRate","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.OperatingFrequencyBand","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.ChannelsInUse","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Channel","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.AutoChannelEnable","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.OperatingChannelBandwidth","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.RadioResetCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.PacketsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.PacketsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.ErrorsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.ErrorsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.DiscardPacketsSent","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.DiscardPacketsReceived","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.PLCPErrorCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.FCSErrorCount","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_NoiseFloor","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.Noise","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ChannelUtilization","type":"dataModel"},{"reference":"Device.WiFi.Radio.2.Stats.X_COMCAST-COM_ActivityFactor","type":"dataModel"}],"Protocol":"HTTP","ReportingInterval":60,"TimeReference":"0001-01-01T00:00:00Z","Version":"0.1"},"versionHash":"ed0de6ef"}]`

	appendedBytes, err := AppendProfiles([]byte(mockedBaseProfilesResponse), []byte(mockedExtraProfilesResponse))
	assert.NilError(t, err)

	var itf interface{}
	err = json.Unmarshal(appendedBytes, &itf)
	assert.NilError(t, err)
	_, err = json.MarshalIndent(itf, "", "  ")
	assert.NilError(t, err)
}
