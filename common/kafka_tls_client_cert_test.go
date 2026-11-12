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
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-akka/configuration"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/assert"
)

type testCertPair struct {
	certPEM []byte
	keyPEM  []byte
}

func newTestCertPair(t *testing.T, dnsNames ...string) testCertPair {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              dnsNames,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	assert.NilError(t, err)

	return testCertPair{
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		keyPEM:  pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}),
	}
}

// combined returns a single PEM containing the certificate followed by the private key.
func (p testCertPair) combined() []byte {
	out := append([]byte{}, p.certPEM...)
	return append(out, p.keyPEM...)
}

func writeTestFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	assert.NilError(t, os.WriteFile(path, data, 0600))
	return path
}

// quoted wraps a path for use as a HOCON string value.
func quoted(path string) string {
	return `"` + path + `"`
}

// loadKafkaTLSForTest parses body as the contents of webconfig.kafka and loads its TLS config.
func loadKafkaTLSForTest(body string) (*tls.Config, error) {
	conf := configuration.ParseString("webconfig { kafka {\n" + body + "\n} }")
	return LoadKafkaTLSConfig(conf, "webconfig.kafka")
}

func TestKafkaTLSClientCert_DisabledIgnoresClientCert(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "client.pem", pair.combined())

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = false\ntls_cert_file = " + quoted(certFile))

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig == nil, "TLS config should be nil when TLS is disabled")
}

func TestKafkaTLSClientCert_CombinedPEM(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "client-combined.pem", pair.combined())

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile))

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil)
	assert.Equal(t, len(tlsConfig.Certificates), 1)
	assert.Assert(t, tlsConfig.Certificates[0].PrivateKey != nil, "Combined PEM should yield a private key")
}

func TestKafkaTLSClientCert_SeparateCertAndKey(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "client.crt", pair.certPEM)
	keyFile := writeTestFile(t, "client.key", pair.keyPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile) + "\ntls_key_file = " + quoted(keyFile))

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig != nil)
	assert.Equal(t, len(tlsConfig.Certificates), 1)
	assert.Assert(t, tlsConfig.Certificates[0].PrivateKey != nil, "Separate key should yield a private key")
}

func TestKafkaTLSClientCert_CertificateWithoutPrivateKey(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "cert-only.pem", pair.certPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile))

	assert.Assert(t, err != nil, "Certificate without a private key should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to load TLS certificate and key")
}

func TestKafkaTLSClientCert_InvalidCertificateInCombinedPEM(t *testing.T) {
	pair := newTestCertPair(t)
	garbage := []byte("-----BEGIN CERTIFICATE-----\nbm90IGEgY2VydGlmaWNhdGU=\n-----END CERTIFICATE-----\n")
	combined := append(garbage, pair.keyPEM...)
	certFile := writeTestFile(t, "invalid-combined.pem", combined)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile))

	assert.Assert(t, err != nil, "Invalid certificate should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to load TLS certificate")
}

func TestKafkaTLSClientCert_InvalidPrivateKey(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "client.crt", pair.certPEM)
	keyFile := writeTestFile(t, "invalid.key", []byte("not a private key"))

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile) + "\ntls_key_file = " + quoted(keyFile))

	assert.Assert(t, err != nil, "Invalid private key should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to load TLS certificate and key")
}

func TestKafkaTLSClientCert_MismatchedCertAndKey(t *testing.T) {
	pairA := newTestCertPair(t)
	pairB := newTestCertPair(t)
	certFile := writeTestFile(t, "client.crt", pairA.certPEM)
	keyFile := writeTestFile(t, "other.key", pairB.keyPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile) + "\ntls_key_file = " + quoted(keyFile))

	assert.Assert(t, err != nil, "Mismatched certificate and key should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "does not match")
}

func TestKafkaTLSClientCert_MissingSeparateKeyFile(t *testing.T) {
	pair := newTestCertPair(t)
	certFile := writeTestFile(t, "client.crt", pair.certPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = " + quoted(certFile) + "\ntls_key_file = \"/nonexistent/client.key\"")

	assert.Assert(t, err != nil, "Missing key file should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "TLS key file does not exist")
}

func TestKafkaTLSClientCert_MissingCertFile(t *testing.T) {
	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_cert_file = \"/nonexistent/client.pem\"")

	assert.Assert(t, err != nil, "Missing certificate file should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "TLS certificate file does not exist")
}

func TestKafkaTLSClientCert_KeyWithoutCertificate(t *testing.T) {
	pair := newTestCertPair(t)
	keyFile := writeTestFile(t, "client.key", pair.keyPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_key_file = " + quoted(keyFile))

	assert.Assert(t, err != nil, "Key file without certificate file should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "configured without tls_cert_file")
}

// Explicitly configured client credentials must never be dropped, even when verification is skipped.
func TestKafkaTLSClientCert_InvalidCredentialsDoNotDowngradeTLS(t *testing.T) {
	garbageCert := writeTestFile(t, "garbage.pem", []byte("garbage"))
	missingCert := "/nonexistent/client.pem"
	pair := newTestCertPair(t)
	caFile := writeTestFile(t, "ca.crt", pair.certPEM)

	cases := map[string]string{
		"invalid combined pem":    "tls_cert_file = " + quoted(garbageCert),
		"missing certificate":     "tls_cert_file = " + quoted(missingCert),
		"key without certificate": "tls_key_file = " + quoted(missingCert),
	}
	for name, settings := range cases {
		for _, insecure := range []string{"false", "true"} {
			t.Run(fmt.Sprintf("%s/insecure=%s", name, insecure), func(t *testing.T) {
				body := "tls_enabled = true\ntls_insecure_skip_verify = " + insecure + "\ntls_ca_file = " + quoted(caFile) + "\n" + settings
				tlsConfig, err := loadKafkaTLSForTest(body)

				assert.Assert(t, err != nil, "invalid client credentials must fail instead of falling back to server-only TLS")
				assert.Assert(t, tlsConfig == nil)
			})
		}
	}
}

func TestKafkaTLSClientCert_LegacyCANameLoadsCA(t *testing.T) {
	pair := newTestCertPair(t)
	caFile := writeTestFile(t, "ca.crt", pair.certPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_ca_cert_file = " + quoted(caFile))

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig.RootCAs != nil, "legacy tls_ca_cert_file must still configure the broker CA")
}

func TestKafkaTLSClientCert_BothCANamesSameValue(t *testing.T) {
	pair := newTestCertPair(t)
	caFile := writeTestFile(t, "ca.crt", pair.certPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_ca_file = " + quoted(caFile) + "\ntls_ca_cert_file = " + quoted(caFile))

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig.RootCAs != nil)
}

func TestKafkaTLSClientCert_BothCANamesConflictingValues(t *testing.T) {
	pair := newTestCertPair(t)
	preferredCA := writeTestFile(t, "preferred-ca.crt", pair.certPEM)
	legacyCA := writeTestFile(t, "legacy-ca.crt", pair.certPEM)

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_ca_file = " + quoted(preferredCA) + "\ntls_ca_cert_file = " + quoted(legacyCA))

	assert.Assert(t, err != nil, "conflicting CA names must not be resolved silently")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "both set to different values")
}

// Mirrors the Kafka block in the deployed environment configs, which uses the legacy CA name.
func TestKafkaTLSClientCert_DeployedEnvironmentShape(t *testing.T) {
	pair := newTestCertPair(t)
	caFile := writeTestFile(t, "CA_Comodo_Sectigo_xPKI.crt", pair.certPEM)

	body := "tls_enabled = true\ntls_insecure_skip_verify = false\ntls_ca_cert_file = " + quoted(caFile) + "\ntls_server_name = \"*.xdp.comcast.net\""
	tlsConfig, err := loadKafkaTLSForTest(body)

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig.RootCAs != nil)
	assert.Equal(t, tlsConfig.ServerName, "*.xdp.comcast.net")
	assert.Equal(t, tlsConfig.InsecureSkipVerify, false)
	assert.Equal(t, len(tlsConfig.Certificates), 0, "deployed config has no client certificate")
}

func TestKafkaTLSClientCert_InvalidCAFile(t *testing.T) {
	caFile := writeTestFile(t, "ca.crt", []byte("not a certificate"))

	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_ca_file = " + quoted(caFile))

	assert.Assert(t, err != nil, "Invalid CA file should fail")
	assert.Assert(t, tlsConfig == nil)
	assert.ErrorContains(t, err, "failed to parse TLS CA certificate")
}

func TestKafkaTLSClientCert_ServerNameAndVerification(t *testing.T) {
	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_server_name = \"broker.example.com\"")

	assert.NilError(t, err)
	assert.Equal(t, tlsConfig.ServerName, "broker.example.com")
	assert.Assert(t, !tlsConfig.InsecureSkipVerify, "Verification must stay enabled by default")
	assert.Assert(t, len(tlsConfig.Certificates) == 0, "No client certificate should be configured")
}

func TestKafkaTLSClientCert_InsecureSkipVerifyWithoutClientCert(t *testing.T) {
	tlsConfig, err := loadKafkaTLSForTest("tls_enabled = true\ntls_insecure_skip_verify = true")

	assert.NilError(t, err)
	assert.Assert(t, tlsConfig.InsecureSkipVerify)
	assert.Assert(t, len(tlsConfig.Certificates) == 0)
}

// Verifies that a broker requiring client certificates completes mutual authentication
// when the client's combined PEM is configured, and rejects clients that present none.
func TestKafkaTLSClientCert_MutualHandshake(t *testing.T) {
	serverPair := newTestCertPair(t, "localhost")
	clientPair := newTestCertPair(t)

	serverCert, err := tls.X509KeyPair(serverPair.certPEM, serverPair.keyPEM)
	assert.NilError(t, err)
	clientPool := x509.NewCertPool()
	assert.Assert(t, clientPool.AppendCertsFromPEM(clientPair.certPEM))

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientPool,
		MinVersion:   tls.VersionTLS12,
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	assert.NilError(t, err)
	defer ln.Close()

	serverCAFile := writeTestFile(t, "server-ca.crt", serverPair.certPEM)
	combinedFile := writeTestFile(t, "client-combined.pem", clientPair.combined())

	testCases := []struct {
		name     string
		settings string
		expectOK bool
	}{
		{
			name:     "client presents combined certificate and key",
			settings: "tls_cert_file = " + quoted(combinedFile),
			expectOK: true,
		},
		{
			name:     "client presents no certificate",
			settings: "",
			expectOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serverResult := make(chan error, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					serverResult <- err
					return
				}
				defer conn.Close()
				tlsConn := conn.(*tls.Conn)
				_ = tlsConn.SetDeadline(time.Now().Add(5 * time.Second))
				err = tlsConn.Handshake()
				if err == nil && len(tlsConn.ConnectionState().PeerCertificates) != 1 {
					err = fmt.Errorf("expected one peer certificate, got %d", len(tlsConn.ConnectionState().PeerCertificates))
				}
				serverResult <- err
			}()

			body := "tls_enabled = true\ntls_ca_file = " + quoted(serverCAFile) + "\ntls_server_name = \"localhost\"\n" + tc.settings
			clientTLS, err := loadKafkaTLSForTest(body)
			assert.NilError(t, err)

			clientConn, dialErr := tls.Dial("tcp", ln.Addr().String(), clientTLS)
			if dialErr == nil {
				defer clientConn.Close()
				// TLS 1.3 clients may finish before the server rejects them; a read surfaces the alert.
				_ = clientConn.SetDeadline(time.Now().Add(5 * time.Second))
				_, dialErr = clientConn.Read(make([]byte, 1))
				if errors.Is(dialErr, io.EOF) {
					dialErr = nil
				}
			}

			var serverErr error
			select {
			case serverErr = <-serverResult:
			case <-time.After(10 * time.Second):
				t.Fatal("timed out waiting for server handshake")
			}

			if tc.expectOK {
				assert.NilError(t, dialErr)
				assert.NilError(t, serverErr)
			} else {
				assert.Assert(t, serverErr != nil, "server requiring client certificates must reject a client without one")
			}
		})
	}
}

// Verifies the handshake log reports whether the broker requested a client certificate and whether one was sent.
func TestKafkaTLSHandshakeDialer_LogsClientCertificateStatus(t *testing.T) {
	serverPair := newTestCertPair(t, "localhost")
	clientPair := newTestCertPair(t)

	serverCert, err := tls.X509KeyPair(serverPair.certPEM, serverPair.keyPEM)
	assert.NilError(t, err)
	clientPool := x509.NewCertPool()
	assert.Assert(t, clientPool.AppendCertsFromPEM(clientPair.certPEM))
	serverCAFile := writeTestFile(t, "server-ca.crt", serverPair.certPEM)
	combinedFile := writeTestFile(t, "client-combined.pem", clientPair.combined())

	testCases := []struct {
		name          string
		clientAuth    tls.ClientAuthType
		settings      string
		expectRequest bool
		expectSent    bool
	}{
		{
			name:          "broker requires client certificate and client sends one",
			clientAuth:    tls.RequireAndVerifyClientCert,
			settings:      "tls_cert_file = " + quoted(combinedFile),
			expectRequest: true,
			expectSent:    true,
		},
		{
			name:          "broker requests client certificate but client sends none",
			clientAuth:    tls.RequestClientCert,
			settings:      "",
			expectRequest: true,
			expectSent:    false,
		},
		{
			name:          "broker does not request client certificate",
			clientAuth:    tls.NoClientCert,
			settings:      "tls_cert_file = " + quoted(combinedFile),
			expectRequest: false,
			expectSent:    false,
		},
	}

	hook := logtest.NewGlobal()
	defer hook.Reset()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hook.Reset()
			ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
				Certificates: []tls.Certificate{serverCert},
				ClientAuth:   tc.clientAuth,
				ClientCAs:    clientPool,
				MinVersion:   tls.VersionTLS12,
			})
			assert.NilError(t, err)
			defer ln.Close()

			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.(*tls.Conn).Handshake()
			}()

			body := "tls_enabled = true\ntls_ca_file = " + quoted(serverCAFile) + "\ntls_server_name = \"localhost\"\n" + tc.settings
			clientTLS, err := loadKafkaTLSForTest(body)
			assert.NilError(t, err)

			hook.Reset()
			d := NewKafkaTLSHandshakeDialer(clientTLS, 5*time.Second, 0, nil)
			conn, err := d.Dial("tcp", ln.Addr().String())
			assert.NilError(t, err)
			defer conn.Close()

			entries := hook.AllEntries()
			assert.Equal(t, len(entries), 1)
			assert.Equal(t, entries[0].Message, "TLS handshake completed")
			assert.Equal(t, entries[0].Data["logger"], "kafka")
			assert.Equal(t, entries[0].Data["broker"], ln.Addr().String())
			assert.Equal(t, entries[0].Data["client_certificate_requested"], tc.expectRequest)
			assert.Equal(t, entries[0].Data["client_certificate_sent"], tc.expectSent)
		})
	}
}

// Verifies a failed handshake still logs the client certificate status and the error before the dial returns.
func TestKafkaTLSHandshakeDialer_LogsFailureWithClientCertificateStatus(t *testing.T) {
	serverPair := newTestCertPair(t, "localhost")
	clientPair := newTestCertPair(t)

	serverCert, err := tls.X509KeyPair(serverPair.certPEM, serverPair.keyPEM)
	assert.NilError(t, err)
	clientPool := x509.NewCertPool()
	assert.Assert(t, clientPool.AppendCertsFromPEM(clientPair.certPEM))
	serverCAFile := writeTestFile(t, "server-ca.crt", serverPair.certPEM)

	hook := logtest.NewGlobal()
	defer hook.Reset()

	// TLS 1.2 reports the broker's rejection during the client handshake.
	// TLS 1.3 completes the client handshake first, so the rejection is not seen by the dialer.
	testCases := []struct {
		name        string
		version     uint16
		expectError bool
	}{
		{name: "TLS 1.2 broker rejects missing client certificate", version: tls.VersionTLS12, expectError: true},
		{name: "TLS 1.3 rejection occurs after client handshake completes", version: tls.VersionTLS13, expectError: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hook.Reset()
			ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
				Certificates: []tls.Certificate{serverCert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    clientPool,
				MinVersion:   tc.version,
				MaxVersion:   tc.version,
			})
			assert.NilError(t, err)
			defer ln.Close()

			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.(*tls.Conn).Handshake()
			}()

			body := "tls_enabled = true\ntls_ca_file = " + quoted(serverCAFile) + "\ntls_server_name = \"localhost\""
			clientTLS, err := loadKafkaTLSForTest(body)
			assert.NilError(t, err)
			clientTLS.MaxVersion = tc.version

			hook.Reset()
			d := NewKafkaTLSHandshakeDialer(clientTLS, 5*time.Second, 0, nil)
			conn, err := d.Dial("tcp", ln.Addr().String())
			if conn != nil {
				defer conn.Close()
			}

			entries := hook.AllEntries()
			assert.Equal(t, len(entries), 1)
			assert.Equal(t, entries[0].Data["logger"], "kafka")
			assert.Equal(t, entries[0].Data["broker"], ln.Addr().String())
			assert.Equal(t, entries[0].Data["client_certificate_requested"], true)
			assert.Equal(t, entries[0].Data["client_certificate_sent"], false)

			if tc.expectError {
				assert.Assert(t, err != nil, "dial should fail when the broker rejects the handshake")
				assert.Equal(t, entries[0].Message, "TLS handshake failed")
				assert.Assert(t, entries[0].Data["error"] != nil, "failure log must include the original error")
			} else {
				assert.NilError(t, err)
				assert.Equal(t, entries[0].Message, "TLS handshake completed")
			}
		})
	}
}

// Verifies a refused TCP connection is logged as a broker connection failure, not a TLS handshake failure.
func TestKafkaTLSHandshakeDialer_TCPDialFailureIsNotHandshakeFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	addr := ln.Addr().String()
	assert.NilError(t, ln.Close())

	hook := logtest.NewGlobal()
	defer hook.Reset()

	d := NewKafkaTLSHandshakeDialer(&tls.Config{MinVersion: tls.VersionTLS12}, 2*time.Second, 0, nil)
	conn, err := d.Dial("tcp", addr)
	assert.Assert(t, err != nil, "dial to a closed port must fail")
	assert.Assert(t, conn == nil)

	entries := hook.AllEntries()
	assert.Equal(t, len(entries), 1)
	assert.Equal(t, entries[0].Message, "Kafka broker connection failed")
	assert.Equal(t, entries[0].Data["broker"], addr)
	assert.Equal(t, entries[0].Data["client_certificate_requested"], false)
	assert.Equal(t, entries[0].Data["client_certificate_sent"], false)
}
