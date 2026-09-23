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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
)

// LoadTLSConfig loads the shared flat tls_* configuration used by Kafka and Cassandra.
// Prefix should be empty for a driver sub-config, or a full HOCON path such as
// "webconfig.kafka" or "webconfig.kafka_producer".
func LoadTLSConfig(conf *configuration.Config, prefix, component string) (*tls.Config, error) {
	key := func(name string) string {
		if prefix == "" {
			return name
		}
		return prefix + "." + name
	}

	insecureSkipVerify := conf.GetBoolean(key("tls_insecure_skip_verify"))
	certFile := conf.GetString(key("tls_cert_file"))
	keyFile := conf.GetString(key("tls_key_file"))
	caCertFile := conf.GetString(key("tls_ca_cert_file"))
	serverName := conf.GetString(key("tls_server_name"))

	cipherSuites := []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
	if component == "Cassandra" {
		cipherSuites = append(cipherSuites, tls.TLS_RSA_WITH_AES_128_CBC_SHA)
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverName,
		CipherSuites:       cipherSuites,
		InsecureSkipVerify: insecureSkipVerify,
	}

	if insecureSkipVerify && (len(certFile) == 0 || len(keyFile) == 0) {
		log.WithField("component", component).Warn("TLS enabled in insecure mode without client certificates")
	} else if len(certFile) > 0 && len(keyFile) > 0 {
		if !insecureSkipVerify {
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS certificate file does not exist: %s", certFile))
			}
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS key file does not exist: %s", keyFile))
			}
		}

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to load TLS certificate and key from %s and %s: %v", certFile, keyFile, err))
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		log.WithFields(log.Fields{
			"component": component,
			"cert_file": certFile,
			"key_file":  keyFile,
		}).Info("Loaded TLS client certificate for mTLS")
	} else if len(certFile) > 0 || len(keyFile) > 0 {
		if !insecureSkipVerify {
			return nil, NewError(fmt.Errorf("TLS enabled with verification but incomplete certificate configuration (cert: %s, key: %s)", certFile, keyFile))
		}
	}

	if insecureSkipVerify && len(caCertFile) == 0 {
		log.WithField("component", component).Warn("TLS enabled in insecure mode without CA certificate")
	} else if len(caCertFile) > 0 {
		if !insecureSkipVerify {
			if _, err := os.Stat(caCertFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS CA certificate file does not exist: %s", caCertFile))
			}
		}

		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to read TLS CA certificate from %s: %v", caCertFile, err))
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, NewError(fmt.Errorf("failed to parse TLS CA certificate from %s", caCertFile))
		}
		tlsConfig.RootCAs = caCertPool
		log.WithFields(log.Fields{
			"component":    component,
			"ca_cert_file": caCertFile,
		}).Info("Loaded TLS CA certificate for server verification")
	}

	if insecureSkipVerify {
		log.WithField("component", component).Warn("TLS certificate verification is disabled (insecure_skip_verify=true). This is insecure and should only be used for testing.")
	}
	log.WithFields(log.Fields{
		"component":            component,
		"has_client_cert":      len(tlsConfig.Certificates) > 0,
		"has_ca_cert":          tlsConfig.RootCAs != nil,
		"insecure_skip_verify": insecureSkipVerify,
		"server_name":          serverName,
	}).Info("TLS configuration loaded")

	return tlsConfig, nil
}

// LoadKafkaTLSConfig loads enabled Kafka TLS configuration from HOCON.
// Prefix should be like "webconfig.kafka", "webconfig.kafka.clusters.mesh",
// or "webconfig.kafka_producer".
func LoadKafkaTLSConfig(conf *configuration.Config, prefix string) (*tls.Config, error) {
	tlsEnabled := conf.GetBoolean(prefix + ".tls_enabled")
	if !tlsEnabled {
		return nil, nil
	}
	return LoadTLSConfig(conf, prefix, "Kafka")
}
