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
	"os"
	"strings"

	"github.com/go-akka/configuration"
)

type ServerConfig struct {
	*configuration.Config
	configBytes []byte
}

func NewServerConfig(configFile string) (*ServerConfig, error) {
	configBytes, err := os.ReadFile(configFile)
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

// NOTE that "bad" entries (keys without values, ill-formatted) can still be added
// hence no parsing error
func (c *ServerConfig) AddConfig(args ...string) {
	lines := []string{
		string(c.configBytes),
	}
	lines = append(lines, args...)
	ss := strings.Join(lines, "\n")
	c.Config = configuration.ParseString(ss)
	c.configBytes = []byte(ss)
}

// copy the config and add extra items
func (c *ServerConfig) Copy(args ...string) *ServerConfig {
	lines := []string{
		string(c.configBytes),
	}
	lines = append(lines, args...)
	ss := strings.Join(lines, "\n")
	conf := configuration.ParseString(ss)
	return &ServerConfig{
		Config:      conf,
		configBytes: []byte(ss),
	}
}

func (c *ServerConfig) KafkaClusterNames() []string {
	clustersNodeValue := c.GetNode("webconfig.kafka.clusters")
	if clustersNodeValue == nil {
		return nil
	}

	clustersNode := clustersNodeValue.GetObject()
	return clustersNode.GetKeys()
}
