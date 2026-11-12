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
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// clientCertStatus records what the broker requested and what this client sent during one handshake.
type clientCertStatus struct {
	requested bool
	sent      bool
}

// connectError marks a failure before the TLS handshake starts (TCP dial or address parsing),
// so it is not reported as a handshake failure.
type connectError struct {
	err error
}

func (e connectError) Error() string { return e.err.Error() }

func (e connectError) Unwrap() error { return e.err }

// KafkaTLSHandshakeDialer performs the TLS handshake for each Kafka broker connection itself. sarama
// wraps connections with tls.Client internally and does not expose the handshake result, so the
// handshake is done here to observe whether a client certificate was requested and sent.
// It must be used as sarama's Net.Proxy.Dialer with Net.TLS left disabled, otherwise TLS is applied twice.
type KafkaTLSHandshakeDialer struct {
	netDialer *net.Dialer
	tlsConfig *tls.Config
	timeout   time.Duration
}

// NewKafkaTLSHandshakeDialer takes the sarama net settings so connection behavior matches sarama's own dialer.
func NewKafkaTLSHandshakeDialer(tlsConfig *tls.Config, timeout, keepAlive time.Duration, localAddr net.Addr) *KafkaTLSHandshakeDialer {
	return &KafkaTLSHandshakeDialer{
		netDialer: &net.Dialer{Timeout: timeout, KeepAlive: keepAlive, LocalAddr: localAddr},
		tlsConfig: tlsConfig,
		timeout:   timeout,
	}
}

// Dial is called by sarama, which does not pass a context.
func (d *KafkaTLSHandshakeDialer) Dial(network, addr string) (net.Conn, error) {
	ctx := context.Background()
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	return d.DialContext(ctx, network, addr)
}

func (d *KafkaTLSHandshakeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, status, err := d.dialTLS(ctx, network, addr)
	fields := log.Fields{
		"logger":                       "kafka",
		"broker":                       addr,
		"client_certificate_requested": status.requested,
		"client_certificate_sent":      status.sent,
	}
	if err != nil {
		var connErr connectError
		if errors.As(err, &connErr) {
			log.WithFields(fields).WithError(err).Error("Kafka broker connection failed")
		} else {
			log.WithFields(fields).WithError(err).Error("TLS handshake failed")
		}
		return nil, err
	}
	log.WithFields(fields).Info("TLS handshake completed")
	return conn, nil
}

func (d *KafkaTLSHandshakeDialer) dialTLS(ctx context.Context, network, addr string) (net.Conn, clientCertStatus, error) {
	var status clientCertStatus
	raw, err := d.netDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, status, connectError{err}
	}

	cfg := d.tlsConfig.Clone()
	if cfg.ServerName == "" {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			raw.Close()
			return nil, status, connectError{err}
		}
		cfg.ServerName = host
	}
	// Same certificate selection as crypto/tls when GetClientCertificate is unset: the first
	// configured certificate that matches the broker's request, or an empty certificate.
	cfg.GetClientCertificate = func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		status.requested = true
		for i := range cfg.Certificates {
			if err := cri.SupportsCertificate(&cfg.Certificates[i]); err == nil {
				status.sent = true
				return &cfg.Certificates[i], nil
			}
		}
		return new(tls.Certificate), nil
	}

	conn := tls.Client(raw, cfg)
	if err := conn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, status, err
	}
	return conn, status, nil
}
