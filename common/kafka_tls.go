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
	caFile := conf.GetString(key("tls_ca_file"))
	serverName := conf.GetString(key("tls_server_name"))

	// Fail loudly rather than silently ignoring the CA and falling back to the system pool.
	if len(conf.GetString(key("tls_ca_cert_file"))) > 0 {
		return nil, NewError(fmt.Errorf("%s: tls_ca_cert_file is no longer supported; rename it to tls_ca_file", component))
	}

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

	if len(certFile) > 0 {
		// Combined PEM: when no separate key is configured, the private key is read from tls_cert_file.
		keySource := keyFile
		if len(keySource) == 0 {
			keySource = certFile
		} else if err := checkFileExists(keySource, "TLS key"); err != nil {
			return nil, err
		}
		if err := checkFileExists(certFile, "TLS certificate"); err != nil {
			return nil, err
		}

		cert, err := tls.LoadX509KeyPair(certFile, keySource)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to load TLS certificate and key from %s and %s (without tls_key_file, the private key must be in tls_cert_file): %v", certFile, keySource, err))
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		log.WithFields(log.Fields{
			"component": component,
			"cert_file": certFile,
			"key_file":  keySource,
		}).Info("Loaded TLS client certificate")
	} else if len(keyFile) > 0 {
		return nil, NewError(fmt.Errorf("TLS key file %s is configured without tls_cert_file", keyFile))
	}

	if len(caFile) > 0 {
		if err := checkFileExists(caFile, "TLS CA certificate"); err != nil {
			return nil, err
		}

		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, NewError(fmt.Errorf("failed to read TLS CA certificate from %s: %v", caFile, err))
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, NewError(fmt.Errorf("failed to parse TLS CA certificate from %s", caFile))
		}
		tlsConfig.RootCAs = caCertPool
		log.WithFields(log.Fields{
			"component":    component,
			"ca_cert_file": caFile,
		}).Info("Loaded TLS CA certificate for server verification")
	}

	if insecureSkipVerify {
		log.WithField("component", component).Warn("TLS certificate verification is disabled (tls_insecure_skip_verify=true). This is insecure and should only be used for testing.")
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

// checkFileExists returns a clear configuration error when path does not exist.
func checkFileExists(path, description string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewError(fmt.Errorf("%s file does not exist: %s", description, path))
	}
	return nil
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
