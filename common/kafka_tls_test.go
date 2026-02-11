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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-akka/configuration"
	"gotest.tools/assert"
)

// Helper function to generate a self-signed certificate for testing
func generateTestCertificate(t *testing.T, certFile, keyFile string) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	assert.NilError(t, err)

	// Write certificate to file
	certOut, err := os.Create(certFile)
	assert.NilError(t, err)
	defer certOut.Close()
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	assert.NilError(t, err)

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	assert.NilError(t, err)
	defer keyOut.Close()
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})
	assert.NilError(t, err)
}

func TestLoadKafkaTLSConfig_Disabled(t *testing.T) {
	confStr := `
webconfig {
	kafka {
		tls_enabled = false
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig == nil, "TLS config should be nil when TLS is disabled")
}

func TestLoadKafkaTLSConfig_EnabledWithoutCerts(t *testing.T) {
	confStr := `
webconfig {
	kafka {
		tls_enabled = true
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil when TLS is enabled")
	assert.Assert(t, len(tlsConfig.Certificates) == 0, "Should have no client certificates")
	assert.Assert(t, tlsConfig.RootCAs == nil, "Should have no custom CA")
	assert.Assert(t, !tlsConfig.InsecureSkipVerify, "Should not skip verification by default")
}

func TestLoadKafkaTLSConfig_WithClientCertificates(t *testing.T) {
	// Create temp directory for test certificates
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "client.crt")
	keyFile := filepath.Join(tempDir, "client.key")

	// Generate test certificate
	generateTestCertificate(t, certFile, keyFile)

	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_cert_file = "` + certFile + `"
		tls_key_file = "` + keyFile + `"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil")
	assert.Assert(t, len(tlsConfig.Certificates) == 1, "Should have one client certificate")
	assert.Assert(t, tlsConfig.RootCAs == nil, "Should have no custom CA")
}

func TestLoadKafkaTLSConfig_WithCACertificate(t *testing.T) {
	// Create temp directory for test certificates
	tempDir := t.TempDir()
	caCertFile := filepath.Join(tempDir, "ca.crt")
	caKeyFile := filepath.Join(tempDir, "ca.key")

	// Generate CA certificate
	generateTestCertificate(t, caCertFile, caKeyFile)

	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_ca_cert_file = "` + caCertFile + `"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil")
	assert.Assert(t, len(tlsConfig.Certificates) == 0, "Should have no client certificates")
	assert.Assert(t, tlsConfig.RootCAs != nil, "Should have custom CA")
}

func TestLoadKafkaTLSConfig_WithInsecureSkipVerify(t *testing.T) {
	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_insecure_skip_verify = true
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil")
	assert.Assert(t, tlsConfig.InsecureSkipVerify, "Should skip verification when configured")
}

func TestLoadKafkaTLSConfig_MissingCertFile(t *testing.T) {
	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_cert_file = "/nonexistent/client.crt"
		tls_key_file = "/nonexistent/client.key"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.Assert(t, err != nil, "Should return error for missing certificate file")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "does not exist")
}

func TestLoadKafkaTLSConfig_MissingCertFileWithInsecureSkipVerify(t *testing.T) {
	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_insecure_skip_verify = true
		tls_cert_file = "/nonexistent/client.crt"
		tls_key_file = "/nonexistent/client.key"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	// When insecure_skip_verify is true, missing cert files should not cause validation errors on file check,
	// but will still fail when trying to actually load them
	assert.Assert(t, err != nil, "Should return error trying to load non-existent files")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to load TLS certificate")
}

func TestLoadKafkaTLSConfig_InvalidCertFile(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "invalid.crt")
	keyFile := filepath.Join(tempDir, "invalid.key")

	// Create invalid certificate files
	err := os.WriteFile(certFile, []byte("invalid certificate content"), 0644)
	assert.NilError(t, err)
	err = os.WriteFile(keyFile, []byte("invalid key content"), 0644)
	assert.NilError(t, err)

	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_cert_file = "` + certFile + `"
		tls_key_file = "` + keyFile + `"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.Assert(t, err != nil, "Should return error for invalid certificate file")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to load TLS certificate")
}

func TestLoadKafkaTLSConfig_MissingCACertFile(t *testing.T) {
	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_ca_cert_file = "/nonexistent/ca.crt"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.Assert(t, err != nil, "Should return error for missing CA certificate file")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "does not exist")
}

func TestLoadKafkaTLSConfig_MissingCACertFileWithInsecureSkipVerify(t *testing.T) {
	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_insecure_skip_verify = true
		tls_ca_cert_file = "/nonexistent/ca.crt"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	// When insecure_skip_verify is true, missing CA file will not cause validation error on file check,
	// but will still fail when trying to actually read it
	assert.Assert(t, err != nil, "Should return error trying to load non-existent CA file")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to read TLS CA certificate")
}

func TestLoadKafkaTLSConfig_InvalidCACertFile(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()
	caCertFile := filepath.Join(tempDir, "invalid_ca.crt")

	// Create invalid CA certificate file
	err := os.WriteFile(caCertFile, []byte("invalid CA certificate content"), 0644)
	assert.NilError(t, err)

	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_ca_cert_file = "` + caCertFile + `"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.Assert(t, err != nil, "Should return error for invalid CA certificate file")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to parse TLS CA certificate")
}

func TestLoadKafkaTLSConfig_FullConfiguration(t *testing.T) {
	// Create temp directory for test certificates
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "client.crt")
	keyFile := filepath.Join(tempDir, "client.key")
	caCertFile := filepath.Join(tempDir, "ca.crt")
	caKeyFile := filepath.Join(tempDir, "ca.key")

	// Generate test certificates
	generateTestCertificate(t, certFile, keyFile)
	generateTestCertificate(t, caCertFile, caKeyFile)

	confStr := `
webconfig {
	kafka {
			tls_enabled = true
		tls_cert_file = "` + certFile + `"
		tls_key_file = "` + keyFile + `"
		tls_ca_cert_file = "` + caCertFile + `"
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil")
	assert.Assert(t, len(tlsConfig.Certificates) == 1, "Should have one client certificate")
	assert.Assert(t, tlsConfig.RootCAs != nil, "Should have custom CA")
	assert.Assert(t, !tlsConfig.InsecureSkipVerify, "Should not skip verification")
}

func TestLoadKafkaTLSConfig_DifferentPrefixes(t *testing.T) {
	// Test with producer prefix
	confStr := `
webconfig {
	kafka_producer {
			tls_enabled = true
		tls_insecure_skip_verify = true
	}
}
`
	conf := configuration.ParseString(confStr)
	tlsConfig, err := LoadKafkaTLSConfig(conf, "webconfig.kafka_producer")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil, "TLS config should not be nil")
	assert.Assert(t, tlsConfig.InsecureSkipVerify)

	// Test with cluster prefix
	confStr2 := `
webconfig {
	kafka {
		clusters {
			mesh {
				tls_enabled = true
			}
		}
	}
}
`
	conf2 := configuration.ParseString(confStr2)
	tlsConfig2, err := LoadKafkaTLSConfig(conf2, "webconfig.kafka.clusters.mesh")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig2 != nil, "TLS config should not be nil for cluster")
}
