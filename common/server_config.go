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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-akka/configuration"
)

var (
	testServerConfig *ServerConfig
	testConfigFiles  = []string{
		"/app/webconfig/test_webconfig.conf",
		"../config/sample_webconfig.conf",
		"/app/webconfig/webconfig.conf",
	}
)

type ServerConfig struct {
	*configuration.Config
	configBytes []byte
}

func NewServerConfig(configFile string) (*ServerConfig, error) {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, NewError(err)
	}
	conf := configuration.ParseString(string(configBytes))
	return &ServerConfig{
		Config:      conf,
		configBytes: configBytes,
	}, nil
}

func (c *ServerConfig) ConfigBytes() []byte {
	return c.configBytes
}

func (c *ServerConfig) KafkaClusterNames() []string {
	clustersNodeValue := c.GetNode("webconfig.kafka.clusters")
	if clustersNodeValue == nil {
		return nil
	}

	clustersNode := clustersNodeValue.GetObject()
	return clustersNode.GetKeys()
}

func GetTestConfigFile() (string, error) {
	testConfigFile := os.Getenv("TEST_CONFIG_FILE")
	if len(testConfigFile) > 0 {
		if _, err := os.Stat(testConfigFile); err == nil {
			return testConfigFile, nil
		}
	}

	for _, cf := range testConfigFiles {
		if _, err := os.Stat(cf); os.IsNotExist(err) {
			continue
		}
		return cf, nil
	}
	return "", NewError(fmt.Errorf("Cannot find any predefined config file for test"))
}

// REMINDER
// this is called from mutiple packages, but we only init the client/session once
func GetTestServerConfig(args ...string) (*ServerConfig, error) {
	if len(args) > 0 {
		c, err := NewServerConfig(args[0])
		if err != nil {
			return nil, NewError(err)
		}
		return c, nil
	}

	if testServerConfig != nil {
		return testServerConfig, nil
	}

	configFile, err := GetTestConfigFile()
	if err != nil {
		return nil, NewError(err)
	}

	// init shared objects
	testServerConfig, err = NewServerConfig(configFile)
	if err != nil {
		return nil, NewError(err)
	}
	return testServerConfig, nil
}
