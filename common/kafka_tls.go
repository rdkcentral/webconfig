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

// LoadKafkaTLSConfig loads TLS configuration from HOCON config at the specified prefix.
// Prefix should be like "webconfig.kafka" or "webconfig.kafka.clusters.mesh" or "webconfig.kafka_producer".
// Returns nil if TLS is not enabled.
// Returns error if TLS is enabled but configuration is invalid.
func LoadKafkaTLSConfig(conf *configuration.Config, prefix string) (*tls.Config, error) {
	tlsEnabled := conf.GetBoolean(prefix + ".tls_enabled")
	if !tlsEnabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	// Check insecure_skip_verify flag first
	insecureSkipVerify := conf.GetBoolean(prefix + ".tls_insecure_skip_verify")

	// Load client certificates for mTLS if provided (optional when insecure_skip_verify is true)
	certFile := conf.GetString(prefix + ".tls_cert_file")
	keyFile := conf.GetString(prefix + ".tls_key_file")

	// When insecure_skip_verify is true and no cert files configured, skip loading certificates
	// This allows TLS without client authentication (server-only TLS)
	if insecureSkipVerify && (len(certFile) == 0 || len(keyFile) == 0) {
		// Insecure mode without client certificates - skip cert loading
	} else if len(certFile) > 0 && len(keyFile) > 0 {
		// Only validate cert files exist when verification is enabled
		if !insecureSkipVerify {
			// Validate certificate file exists
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS certificate file does not exist: %s", certFile))
			}

			// Validate key file exists
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS key file does not exist: %s", keyFile))
			}
		}

		// Load and parse the certificate and key
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to load TLS certificate and key from %s and %s: %v", certFile, keyFile, err))
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		log.WithFields(log.Fields{
			"prefix":    prefix,
			"cert_file": certFile,
			"key_file":  keyFile,
		}).Info("Loaded TLS client certificate for mTLS")
	} else if len(certFile) > 0 || len(keyFile) > 0 {
		// Partial cert configuration detected - require both cert and key when verification is enabled
		if !insecureSkipVerify {
			return nil, NewError(fmt.Errorf("TLS enabled with verification but incomplete certificate configuration (cert: %s, key: %s)", certFile, keyFile))
		}
	}

	// Load CA certificate if provided (optional when insecure_skip_verify is true)
	caCertFile := conf.GetString(prefix + ".tls_ca_cert_file")
	// When insecure_skip_verify is true and no CA file configured, skip loading CA cert
	// This allows TLS without broker verification (insecure mode)
	if insecureSkipVerify && len(caCertFile) == 0 {
		// Insecure mode without CA cert - skip CA loading
	} else if len(caCertFile) > 0 {
		// Only validate CA cert file exists when verification is enabled
		if !insecureSkipVerify {
			// Validate CA certificate file exists
			if _, err := os.Stat(caCertFile); os.IsNotExist(err) {
				return nil, NewError(fmt.Errorf("TLS CA certificate file does not exist: %s", caCertFile))
			}
		}

		// Load CA certificate
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to read TLS CA certificate from %s: %v", caCertFile, err))
		}

		// Parse CA certificate
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, NewError(fmt.Errorf("failed to parse TLS CA certificate from %s", caCertFile))
		}

		tlsConfig.RootCAs = caCertPool
		log.WithFields(log.Fields{
			"prefix":       prefix,
			"ca_cert_file": caCertFile,
		}).Info("Loaded TLS CA certificate for broker verification")
	}

	if insecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
		log.WithFields(log.Fields{
			"prefix": prefix,
		}).Warn("TLS certificate verification is disabled (insecure_skip_verify=true). This is insecure and should only be used for testing.")
	}

	log.WithFields(log.Fields{
		"prefix":               prefix,
		"has_client_cert":      len(tlsConfig.Certificates) > 0,
		"has_ca_cert":          tlsConfig.RootCAs != nil,
		"insecure_skip_verify": insecureSkipVerify,
	}).Info("TLS configuration loaded for Kafka connection")

	return tlsConfig, nil
}
