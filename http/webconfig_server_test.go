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
package http

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/security"
	log "github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestApiTokenAuthSecureDefaults(t *testing.T) {
	// Admin write endpoints (document, rootdocument, poke, reference) MUST
	// be guarded by ApiMiddleware out of the box. Regression guard for f003.
	assert.Assert(t, serverApiTokenAuthEnabledDefault, "server API token auth must default to enabled")
	assert.Assert(t, !configApiTokenAuthEnabledDefault, "config API token auth must default to disabled")
	assert.Assert(t, deviceApiTokenAuthEnabledDefault, "device API token auth must default to enabled")
}

func TestConfigEndpointRemainsUnauthenticatedByDefault(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	assert.Assert(t, !server.ConfigApiTokenAuthEnabled())
	router := server.GetRouter(false)

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusOK)
}

func TestConfigEndpointRequiresApiTokenWhenEnabled(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetConfigApiTokenAuthEnabled(true)
	assert.Assert(t, server.ConfigApiTokenAuthEnabled())
	router := server.GetRouter(false)

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

func TestApiMiddlewareSuppressesConfigRequestLogs(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetConfigApiTokenAuthEnabled(true)
	router := server.GetRouter(false)

	var logs strings.Builder
	previousOutput := log.StandardLogger().Out
	previousLevel := log.StandardLogger().Level
	log.SetOutput(&logs)
	log.SetLevel(log.InfoLevel)
	defer func() {
		log.SetOutput(previousOutput)
		log.SetLevel(previousLevel)
	}()

	req, err := http.NewRequest("GET", "/config", nil)
	assert.NilError(t, err)
	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
	assert.Assert(t, !strings.Contains(logs.String(), "Request started"))
	assert.Assert(t, !strings.Contains(logs.String(), "Request finished"))

	logs.Reset()
	req, err = http.NewRequest("GET", "/other", nil)
	assert.NilError(t, err)
	res = ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
	assert.Assert(t, strings.Contains(logs.String(), "Request started"))
	assert.Assert(t, strings.Contains(logs.String(), "Request finished"))
}

func TestApiMiddlewareReturnsUnauthorizedForAuthenticationFailures(t *testing.T) {
	tests := []struct {
		name string
		auth string
		err  error
	}{
		{name: "missing token"},
		{name: "malformed authorization", auth: "Basic credentials"},
		{name: "bearer with no token", auth: "Bearer"},
		{name: "bearer with multiple parts", auth: "Bearer a b"},
		{name: "malformed token", auth: "Bearer malformed", err: fmt.Errorf("malformed token")},
		{name: "invalid token", auth: "Bearer invalid", err: fmt.Errorf("invalid token")},
		{name: "expired token", auth: "Bearer expired", err: fmt.Errorf("token is expired")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebconfigServer(sc, true)
			server.TokenManager = security.NewTokenManager(sc.Config)
			server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
				return false, "", 0, tt.err
			})

			req, err := http.NewRequest("GET", "/", nil)
			assert.NilError(t, err)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			res := ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("authentication failure reached the next handler")
			}))).Result()
			assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
		})
	}
}

func TestApiMiddlewareReturnsForbiddenForInsufficientCapabilities(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
		return false, "", 0, common.ErrNoCapabilities
	})

	req, err := http.NewRequest("GET", "/", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer authenticated-but-not-capable")
	res := ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authorization failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
}

// XPC-43727 an actually malformed JWT (not a mocked verifier error) must
// still be classified as 401 through the real TokenManager.VerifyToken path
func TestApiMiddlewareRejectsActualMalformedJwt(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)

	req, err := http.NewRequest("GET", "/", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer this-is-not-a-jwt")
	res := ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("malformed jwt reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

func newJwksHttpTestFixture(t *testing.T) (*rsa.PrivateKey, *keyfunc.JWKS) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	const kid = "jwks-http-test-key"
	b64 := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}
	eBytes := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()
	jwkJSON := fmt.Sprintf(`{"keys":[{"kty":"RSA","kid":%q,"alg":"RS256","n":%q,"e":%q}]}`,
		kid, b64(privateKey.PublicKey.N.Bytes()), b64(eBytes))

	jwks, err := keyfunc.NewJSON(json.RawMessage(jwkJSON))
	assert.NilError(t, err)
	return privateKey, jwks
}

func signJwksHttpTestToken(t *testing.T, privateKey *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "jwks-http-test-key"
	tokenString, err := token.SignedString(privateKey)
	assert.NilError(t, err)
	return tokenString
}

// XPC-43727 exercise the ApiMiddleware route actually used when JWKS is
// enabled (server.JwksManager / errors.Is classification via VerifyApiToken),
// not just the underlying JwksManager in isolation
func TestApiMiddlewareJwksEnabledRoute(t *testing.T) {
	privateKey, jwks := newJwksHttpTestFixture(t)
	utcnow := time.Now()

	tests := []struct {
		name   string
		token  string
		status int
	}{
		{
			name: "valid capability",
			token: signJwksHttpTestToken(t, privateKey, jwt.MapClaims{
				"capabilities": []interface{}{"webconfig:all"},
				"exp":          utcnow.Add(time.Hour).Unix(),
			}),
			status: http.StatusOK,
		},
		{
			name: "missing capability",
			token: signJwksHttpTestToken(t, privateKey, jwt.MapClaims{
				"capabilities": []interface{}{"some:other:capability"},
				"exp":          utcnow.Add(time.Hour).Unix(),
			}),
			status: http.StatusForbidden,
		},
		{
			name: "expired token",
			token: signJwksHttpTestToken(t, privateKey, jwt.MapClaims{
				"capabilities": []interface{}{"webconfig:all"},
				"exp":          utcnow.Add(-time.Hour).Unix(),
			}),
			status: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebconfigServer(sc, true)
			server.JwksManager = security.NewJwksManagerFromJWKS(jwks, []string{"webconfig:all"})
			server.SetJwksEnabled(true)

			req, err := http.NewRequest("GET", "/", nil)
			assert.NilError(t, err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tt.token))
			res := ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != http.StatusOK {
					t.Fatal("authentication/authorization failure reached the next handler")
				}
				w.WriteHeader(http.StatusOK)
			}))).Result()
			assert.Equal(t, res.StatusCode, tt.status)
		})
	}
}

// XPC-43727 an invalid-signature token must return 401 through the JWKS
// route, never misclassified as 403
func TestApiMiddlewareJwksEnabledRouteRejectsInvalidSignature(t *testing.T) {
	privateKey, jwks := newJwksHttpTestFixture(t)
	utcnow := time.Now()
	token := signJwksHttpTestToken(t, privateKey, jwt.MapClaims{
		"capabilities": []interface{}{"webconfig:all"},
		"exp":          utcnow.Add(time.Hour).Unix(),
	})

	parts := []rune(token)
	lastDot := -1
	for i, r := range parts {
		if r == '.' {
			lastDot = i
		}
	}
	assert.Assert(t, lastDot >= 0 && lastDot+2 < len(parts))
	mid := lastDot + 1 + (len(parts)-lastDot-1)/2
	if parts[mid] == 'A' {
		parts[mid] = 'B'
	} else {
		parts[mid] = 'A'
	}
	tampered := string(parts)

	server := NewWebconfigServer(sc, true)
	server.JwksManager = security.NewJwksManagerFromJWKS(jwks, []string{"webconfig:all"})
	server.SetJwksEnabled(true)

	req, err := http.NewRequest("GET", "/", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tampered))
	res := ExecuteRequest(req, server.ApiMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("invalid signature reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

func TestCpeMiddlewareReturnsUnauthorizedForInvalidToken(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
		return false, "", 0, fmt.Errorf("invalid token")
	})

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer invalid")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authentication failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

func TestCpeMiddlewareReturnsForbiddenForInsufficientCapabilities(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
		return false, "", 0, common.ErrNoCapabilities
	})

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer authenticated-but-not-capable")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authorization failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
}

func TestCpeMiddlewareReturnsForbiddenForLowTrust(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.SetMinTrust(1000)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
		return true, "comcast", 0, nil
	})

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer authenticated-but-low-trust")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authorization failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
}

// XPC-43727 every class of authentication failure (malformed header,
// malformed/invalid/expired token, mac mismatch) must return 401, never 403.
func TestCpeMiddlewareReturnsUnauthorizedForAuthenticationFailures(t *testing.T) {
	tests := []struct {
		name string
		auth string
		err  error
	}{
		{name: "missing token"},
		{name: "malformed authorization", auth: "Basic credentials"},
		{name: "bearer with no token", auth: "Bearer"},
		{name: "bearer with multiple parts", auth: "Bearer a b"},
		{name: "malformed token", auth: "Bearer malformed", err: fmt.Errorf("malformed token")},
		{name: "invalid signature", auth: "Bearer invalid", err: fmt.Errorf("token signature is invalid")},
		{name: "expired token", auth: "Bearer expired", err: fmt.Errorf("token is expired")},
		{name: "mac mismatch", auth: "Bearer mismatched-mac", err: fmt.Errorf("mac in token does not match claims")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebconfigServer(sc, true)
			server.TokenManager = security.NewTokenManager(sc.Config)
			server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
				return false, "", 0, tt.err
			})

			req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
			assert.NilError(t, err)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
			res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("authentication failure reached the next handler")
			}))).Result()
			assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
		})
	}
}

func TestCpeMiddlewareReturnsBadRequestForMissingMacRouteVar(t *testing.T) {
	server := NewWebconfigServer(sc, true)

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer irrelevant")
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("missing route var reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusBadRequest)
}

// XPC-43727 an actually malformed JWT (not a mocked verifier error) must
// still be classified as 401 through the real TokenManager.VerifyToken path
func TestCpeMiddlewareRejectsActualMalformedJwt(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer this-is-not-a-jwt")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("malformed jwt reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

// XPC-43727 a real signed token for one mac presented against a different
// mac must be rejected as 401, using an in-memory generated key pair so this
// assertion is isolated from external key files/db setup (unlike the
// equivalent case in multipart_test.go's TestCpeMiddleware)
func TestCpeMiddlewareRejectsRealSignedTokenWithWrongMac(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	const kid = "cpe-wrong-mac-test-key"
	decodeKeys := map[string]*rsa.PublicKey{kid: &privateKey.PublicKey}

	sign := func(mac string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"mac": strings.ToLower(mac),
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = kid
		tokenString, err := token.SignedString(privateKey)
		assert.NilError(t, err)
		return tokenString
	}
	token := sign("AABBCCDDEEFF")

	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, validKids []string, requiredCapabilities []string, vargs ...string) (bool, string, int, error) {
		return security.VerifyToken(decodeKeys, []string{kid}, requiredCapabilities, vargs...)
	})

	req, err := http.NewRequest("GET", "/api/v1/device/112233445566/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req = mux.SetURLVars(req, map[string]string{"mac": "112233445566"})
	res := ExecuteRequest(req, server.CpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wrong-mac token reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

func TestTestingCpeMiddlewareReturnsUnauthorizedForMissingToken(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.TestingCpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authentication failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

// XPC-43727 TestingCpeMiddleware independently parses the Authorization
// header and calls VerifyCpeToken directly; verify it classifies the same
// authentication failure classes as CpeMiddleware, despite the separate
// implementation.
func TestTestingCpeMiddlewareReturnsUnauthorizedForAuthenticationFailures(t *testing.T) {
	tests := []struct {
		name string
		auth string
		err  error
	}{
		{name: "malformed authorization", auth: "Basic credentials"},
		{name: "bearer with no token", auth: "Bearer"},
		{name: "bearer with multiple parts", auth: "Bearer a b"},
		{name: "malformed token", auth: "Bearer malformed", err: fmt.Errorf("malformed token")},
		{name: "invalid signature", auth: "Bearer invalid", err: fmt.Errorf("token signature is invalid")},
		{name: "expired token", auth: "Bearer expired", err: fmt.Errorf("token is expired")},
		{name: "mac mismatch", auth: "Bearer mismatched-mac", err: fmt.Errorf("mac in token does not match claims")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebconfigServer(sc, true)
			server.TokenManager = security.NewTokenManager(sc.Config)
			server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
				return false, "", 0, tt.err
			})

			req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
			assert.NilError(t, err)
			req.Header.Set("Authorization", tt.auth)
			req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
			res := ExecuteRequest(req, server.TestingCpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("authentication failure reached the next handler")
			}))).Result()
			assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
		})
	}
}

// XPC-43727 an actually malformed JWT (not a mocked verifier error) must
// still be classified as 401 through the real TokenManager.VerifyToken path
func TestTestingCpeMiddlewareRejectsActualMalformedJwt(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer this-is-not-a-jwt")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.TestingCpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("malformed jwt reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusUnauthorized)
}

// XPC-43727 TestingCpeMiddleware also supports a genuine post-authentication
// authorization failure, distinct from an authentication failure: 403.
func TestTestingCpeMiddlewareReturnsForbiddenForInsufficientCapabilities(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.TokenManager = security.NewTokenManager(sc.Config)
	server.TokenManager.SetVerifyFunc(func(_ map[string]*rsa.PublicKey, _ []string, _ []string, _ ...string) (bool, string, int, error) {
		return false, "", 0, common.ErrNoCapabilities
	})

	req, err := http.NewRequest("GET", "/api/v1/device/AABBCCDDEEFF/config", nil)
	assert.NilError(t, err)
	req.Header.Set("Authorization", "Bearer authenticated-but-not-capable")
	req = mux.SetURLVars(req, map[string]string{"mac": "AABBCCDDEEFF"})
	res := ExecuteRequest(req, server.TestingCpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("authorization failure reached the next handler")
	}))).Result()
	assert.Equal(t, res.StatusCode, http.StatusForbidden)
}

// XPC-43727 TestingCpeMiddleware independently validates the mac route var
// (exact length 12), unlike CpeMiddleware's ValidateMac; verify a malformed
// or missing mac is classified as 400, not 401, when a Bearer token is
// present.
func TestTestingCpeMiddlewareReturnsBadRequestForInvalidMac(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
	}{
		{name: "mac too short", vars: map[string]string{"mac": "AABBCC"}},
		{name: "missing mac var", vars: map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebconfigServer(sc, true)

			req, err := http.NewRequest("GET", "/api/v1/device/x/config", nil)
			assert.NilError(t, err)
			req.Header.Set("Authorization", "Bearer irrelevant")
			req = mux.SetURLVars(req, tt.vars)
			res := ExecuteRequest(req, server.TestingCpeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("invalid mac reached the next handler")
			}))).Result()
			assert.Equal(t, res.StatusCode, http.StatusBadRequest)
		})
	}
}

func TestErrorPreservesUnauthorizedResponse(t *testing.T) {
	recorder := httptest.NewRecorder()

	Error(recorder, http.StatusUnauthorized, nil)

	assert.Equal(t, recorder.Code, http.StatusUnauthorized)
	assert.Equal(t, recorder.Body.Len(), 0)
}

func TestWebconfigServerSetterGetter(t *testing.T) {
	server := NewWebconfigServer(sc, true)

	// factory reset flag
	enabled := true
	server.SetFactoryResetEnabled(enabled)
	assert.Equal(t, server.FactoryResetEnabled(), enabled)
	enabled = false
	server.SetFactoryResetEnabled(enabled)
	assert.Equal(t, server.FactoryResetEnabled(), enabled)

	// server api token auth
	enabled = true
	server.SetServerApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ServerApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetServerApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ServerApiTokenAuthEnabled(), enabled)

	// config api token auth
	enabled = true
	server.SetConfigApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ConfigApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetConfigApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.ConfigApiTokenAuthEnabled(), enabled)

	// device api token auth
	enabled = true
	server.SetDeviceApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.DeviceApiTokenAuthEnabled(), enabled)
	enabled = false
	server.SetDeviceApiTokenAuthEnabled(enabled)
	assert.Equal(t, server.DeviceApiTokenAuthEnabled(), enabled)

	// token api
	enabled = true
	server.SetTokenApiEnabled(enabled)
	assert.Equal(t, server.TokenApiEnabled(), enabled)
	enabled = false
	server.SetTokenApiEnabled(enabled)
	assert.Equal(t, server.TokenApiEnabled(), enabled)

	// kafka
	enabled = true
	server.SetKafkaEnabled(enabled)
	assert.Equal(t, server.KafkaEnabled(), enabled)
	enabled = false
	server.SetKafkaEnabled(enabled)
	assert.Equal(t, server.KafkaEnabled(), enabled)

	// upstream
	enabled = true
	server.SetUpstreamEnabled(enabled)
	assert.Equal(t, server.UpstreamEnabled(), enabled)
	enabled = false
	server.SetUpstreamEnabled(enabled)
	assert.Equal(t, server.UpstreamEnabled(), enabled)

	// app name
	name := "foo"
	server.SetAppName(name)
	assert.Equal(t, server.AppName(), name)
	name = "bar"
	server.SetAppName(name)
	assert.Equal(t, server.AppName(), name)

	// validate mac
	enabled = true
	server.SetValidateMacEnabled(enabled)
	assert.Equal(t, server.ValidateMacEnabled(), enabled)
	enabled = false
	server.SetValidateMacEnabled(enabled)
	assert.Equal(t, server.ValidateMacEnabled(), enabled)

	// validate valid partners
	validPartners := []string{"vendor1", "partner2", "company3"}
	server.SetValidPartners(validPartners)
	assert.DeepEqual(t, server.ValidPartners(), validPartners)
	validPartners = []string{"name3", "name4", "name5"}
	server.SetValidPartners(validPartners)
	assert.DeepEqual(t, server.ValidPartners(), validPartners)

	// get profiles from upstream
	enabled = true
	server.SetUpstreamProfilesEnabled(enabled)
	assert.Equal(t, server.UpstreamProfilesEnabled(), enabled)
	enabled = false
	server.SetUpstreamProfilesEnabled(enabled)
	assert.Equal(t, server.UpstreamProfilesEnabled(), enabled)

	// enforce strict query parameters validation
	enabled = true
	server.SetQueryParamsValidationEnabled(enabled)
	assert.Equal(t, server.QueryParamsValidationEnabled(), enabled)
	enabled = false
	server.SetQueryParamsValidationEnabled(enabled)
	assert.Equal(t, server.QueryParamsValidationEnabled(), enabled)

	//configure trust level
	trust := 1000
	server.SetMinTrust(trust)
	assert.Equal(t, server.MinTrust(), trust)
	trust = 500
	server.SetMinTrust(trust)
	assert.Equal(t, server.MinTrust(), trust)

	x := true
	server.SetFilterOutputByBitmapEnabled(x)
	assert.Assert(t, server.FilterOutputByBitmapEnabled())
	x = false
	server.SetFilterOutputByBitmapEnabled(x)
	assert.Assert(t, !server.FilterOutputByBitmapEnabled())

	validSubdocIdMap := map[string]int{
		"red":    1,
		"orange": 2,
		"yellow": 3,
		"green":  4,
	}
	server.SetValidSubdocIdMap(validSubdocIdMap)
	assert.DeepEqual(t, validSubdocIdMap, server.ValidSubdocIdMap())

}
