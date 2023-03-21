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

// NOTE the codes here are temporary to help debugging some issues.
// They will be removed later to keep the codes generic

const (
	TR181NamePrivatessid = "Device.WiFi.Private"
	TR181NameHomessid    = "Device.WiFi.Home"
)

type EmbeddedSsid struct {
	Ssid      string `json:"SSID" msgpack:"SSID"`
	Enable    bool   `json:"Enable" msgpack:"Enable"`
	Broadcast bool   `json:"SSIDAdvertisementEnabled" msgpack:"SSIDAdvertisementEnabled"`
}

type EmbeddedSecurity struct {
	Passphrase string `json:"Passphrase" msgpack:"Passphrase"`
	Encryption string `json:"EncryptionMethod" msgpack:"EncryptionMethod"`
	Mode       string `json:"ModeEnabled" msgpack:"ModeEnabled"`
}

type EmbeddedPrivateWifi struct {
	Ssid2g      *EmbeddedSsid     `json:"private_ssid_2g,omitempty" msgpack:"private_ssid_2g,omitempty"`
	Ssid5g      *EmbeddedSsid     `json:"private_ssid_5g,omitempty" msgpack:"private_ssid_5g,omitempty"`
	Ssid5gl     *EmbeddedSsid     `json:"private_ssid_5gl,omitempty" msgpack:"private_ssid_5gl,omitempty"`
	Ssid5gu     *EmbeddedSsid     `json:"private_ssid_5gu,omitempty" msgpack:"private_ssid_5gu,omitempty"`
	Ssid6g      *EmbeddedSsid     `json:"private_ssid_6g,omitempty" msgpack:"private_ssid_6g,omitempty"`
	Security2g  *EmbeddedSecurity `json:"private_security_2g,omitempty" msgpack:"private_security_2g,omitempty"`
	Security5g  *EmbeddedSecurity `json:"private_security_5g,omitempty" msgpack:"private_security_5g,omitempty"`
	Security5gl *EmbeddedSecurity `json:"private_security_5gl,omitempty" msgpack:"private_security_5gl,omitempty"`
	Security5gu *EmbeddedSecurity `json:"private_security_5gu,omitempty" msgpack:"private_security_5gu,omitempty"`
	Security6g  *EmbeddedSecurity `json:"private_security_6g,omitempty" msgpack:"private_security_6g,omitempty"`
}

type SimpleWifi struct {
	Ssid2g  *string `json:"ssid_2g,omitempty"`
	Pass2g  *string `json:"pass_2g,omitempty"`
	Mode2g  *string `json:"mode_2g,omitempty"`
	Ssid5g  *string `json:"ssid_5g,omitempty"`
	Pass5g  *string `json:"pass_5g,omitempty"`
	Mode5g  *string `json:"mode_5g,omitempty"`
	Ssid6g  *string `json:"ssid_6g,omitempty"`
	Pass6g  *string `json:"pass_6g,omitempty"`
	Mode6g  *string `json:"mode_6g,omitempty"`
	Version *string `json:"version,omitempty"`
}

func (w *EmbeddedPrivateWifi) GetSimpleWifi(version string) *SimpleWifi {
	var sw SimpleWifi
	if w.Ssid2g != nil {
		sw.Ssid2g = &w.Ssid2g.Ssid
	}
	if w.Ssid5g != nil {
		sw.Ssid5g = &w.Ssid5g.Ssid
	}
	if w.Ssid6g != nil {
		sw.Ssid6g = &w.Ssid6g.Ssid
	}
	if w.Security2g != nil {
		ss := w.Security2g.Passphrase[:4] + "****"
		sw.Pass2g = &ss
		sw.Mode2g = &w.Security2g.Mode
	}
	if w.Security5g != nil {
		ss := w.Security5g.Passphrase[:4] + "****"
		sw.Pass5g = &ss
		sw.Mode5g = &w.Security5g.Mode
	}
	if w.Security6g != nil {
		ss := w.Security6g.Passphrase[:4] + "****"
		sw.Pass6g = &ss
		sw.Mode6g = &w.Security6g.Mode
	}
	sw.Version = &version
	return &sw
}

type EmbeddedHomeWifi struct {
	Ssid2g      *EmbeddedSsid     `json:"home_ssid_2g,omitempty" msgpack:"home_ssid_2g,omitempty"`
	Ssid5g      *EmbeddedSsid     `json:"home_ssid_5g,omitempty" msgpack:"home_ssid_5g,omitempty"`
	Ssid5gl     *EmbeddedSsid     `json:"home_ssid_5gl,omitempty" msgpack:"home_ssid_5gl,omitempty"`
	Ssid5gu     *EmbeddedSsid     `json:"home_ssid_5gu,omitempty" msgpack:"home_ssid_5gu,omitempty"`
	Ssid6g      *EmbeddedSsid     `json:"home_ssid_6g,omitempty" msgpack:"home_ssid_6g,omitempty"`
	Security2g  *EmbeddedSecurity `json:"home_security_2g,omitempty" msgpack:"home_security_2g,omitempty"`
	Security5g  *EmbeddedSecurity `json:"home_security_5g,omitempty" msgpack:"home_security_5g,omitempty"`
	Security5gl *EmbeddedSecurity `json:"home_security_5gl,omitempty" msgpack:"home_security_5gl,omitempty"`
	Security5gu *EmbeddedSecurity `json:"home_security_5gu,omitempty" msgpack:"home_security_5gu,omitempty"`
	Security6g  *EmbeddedSecurity `json:"home_security_6g,omitempty" msgpack:"home_security_6g,omitempty"`
}

func (w *EmbeddedHomeWifi) GetSimpleWifi(version string) *SimpleWifi {
	var sw SimpleWifi
	if w.Ssid2g != nil {
		sw.Ssid2g = &w.Ssid2g.Ssid
	}
	if w.Ssid5g != nil {
		sw.Ssid5g = &w.Ssid5g.Ssid
	}
	if w.Ssid6g != nil {
		sw.Ssid6g = &w.Ssid6g.Ssid
	}
	if w.Security2g != nil {
		ss := w.Security2g.Passphrase[:4] + "****"
		sw.Pass2g = &ss
		sw.Mode2g = &w.Security2g.Mode
	}
	if w.Security5g != nil {
		ss := w.Security5g.Passphrase[:4] + "****"
		sw.Pass5g = &ss
		sw.Mode5g = &w.Security5g.Mode
	}
	if w.Security6g != nil {
		ss := w.Security6g.Passphrase[:4] + "****"
		sw.Pass6g = &ss
		sw.Mode6g = &w.Security6g.Mode
	}
	sw.Version = &version
	return &sw
}
