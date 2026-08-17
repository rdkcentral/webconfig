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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/db"
	"github.com/rdkcentral/webconfig/db/cassandra"
	"github.com/rdkcentral/webconfig/util"
	"gotest.tools/assert"
)

// errSimulatedTimeout is a sentinel used by timeoutMockClient to trigger the 504 path.
var errSimulatedTimeout = errors.New("simulated cassandra timeout")

// timeoutMockClient wraps the real (SQLite) DatabaseClient and overrides IsDbTimeout plus
// selected read methods to return errSimulatedTimeout, simulating a Cassandra timeout
// without a live Cassandra instance.  Overridden methods:
//   - IsDbTimeout     — recognises errSimulatedTimeout as a timeout
//   - GetSubDocument  — used by GetSubDocumentHandler (GET /document/{id})
//   - GetRootDocumentLabels — used by PostSubDocumentHandler (POST /document/{id})
//   - GetRootDocument — used by BuildGetDocument (GET /config)
//   - GetDocument     — used by BuildGetDocument fallback paths
//
// All other interface methods (SetSubDocument, DeleteDocument, etc.) fall through to the
// embedded SQLite client and execute normally.
type timeoutMockClient struct {
	db.DatabaseClient
}

func (m *timeoutMockClient) IsDbTimeout(err error) bool {
	return errors.Is(err, errSimulatedTimeout)
}

func (m *timeoutMockClient) GetSubDocument(mac, subdocId string) (*common.SubDocument, error) {
	return nil, errSimulatedTimeout
}

func (m *timeoutMockClient) GetRootDocumentLabels(mac string) (prometheus.Labels, error) {
	return nil, errSimulatedTimeout
}

func (m *timeoutMockClient) GetRootDocument(mac string) (*common.RootDocument, error) {
	return nil, errSimulatedTimeout
}

func (m *timeoutMockClient) GetDocument(mac string, args ...interface{}) (*common.Document, error) {
	return nil, errSimulatedTimeout
}

// TestIsDbTimeout tests CassandraClient.IsDbTimeout against all gocql timeout error
// variants — including connection-closed and context deadline — as well as direct errors,
// wrapped errors, and multi-layer chains.  Non-timeout errors must return false.
// No live Cassandra connection is required because IsDbTimeout is a pure error-inspection
// function.
func TestIsDbTimeout(t *testing.T) {
	c := &cassandra.CassandraClient{}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrTimeoutNoResponse direct",
			err:  gocql.ErrTimeoutNoResponse,
			want: true,
		},
		{
			name: "ErrTimeoutNoResponse wrapped once",
			err:  fmt.Errorf("layer: %w", gocql.ErrTimeoutNoResponse),
			want: true,
		},
		{
			name: "ErrTimeoutNoResponse wrapped via common.NewError",
			err:  common.NewError(gocql.ErrTimeoutNoResponse),
			want: true,
		},
		{
			name: "ErrTimeoutNoResponse wrapped multiple times",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", gocql.ErrTimeoutNoResponse)),
			want: true,
		},
		{
			name: "RequestErrReadTimeout direct",
			err:  &gocql.RequestErrReadTimeout{},
			want: true,
		},
		{
			name: "RequestErrReadTimeout wrapped",
			err:  fmt.Errorf("layer: %w", &gocql.RequestErrReadTimeout{}),
			want: true,
		},
		{
			name: "RequestErrWriteTimeout direct",
			err:  &gocql.RequestErrWriteTimeout{},
			want: true,
		},
		{
			name: "RequestErrWriteTimeout wrapped",
			err:  fmt.Errorf("layer: %w", &gocql.RequestErrWriteTimeout{}),
			want: true,
		},
		{
			name: "ErrConnectionClosed direct",
			err:  gocql.ErrConnectionClosed,
			want: true,
		},
		{
			name: "ErrConnectionClosed wrapped",
			err:  fmt.Errorf("layer: %w", gocql.ErrConnectionClosed),
			want: true,
		},
		{
			name: "context.DeadlineExceeded direct",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "context.DeadlineExceeded wrapped",
			err:  fmt.Errorf("layer: %w", context.DeadlineExceeded),
			want: true,
		},
		{
			name: "context.Canceled direct",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "context.Canceled wrapped",
			err:  fmt.Errorf("layer: %w", context.Canceled),
			want: false,
		},
		{
			name: "ErrNotFound is not a timeout",
			err:  gocql.ErrNotFound,
			want: false,
		},
		{
			name: "generic error is not a timeout",
			err:  errors.New("some db error"),
			want: false,
		},
		{
			name: "nil is not a timeout",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.IsDbTimeout(tc.err)
			assert.Equal(t, got, tc.want)
		})
	}
}

// TestDbErrToStatus verifies that dbErrToStatus maps a timeout error to 504 and a
// non-timeout error to 500, using the mock client to drive IsDbTimeout.
func TestDbErrToStatus(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.DatabaseClient = &timeoutMockClient{DatabaseClient: server.DatabaseClient}

	assert.Equal(t, server.dbErrToStatus(errSimulatedTimeout), http.StatusGatewayTimeout)
	assert.Equal(t, server.dbErrToStatus(errors.New("other error")), http.StatusInternalServerError)
}

// TestGetSubDocumentHandlerCassandraTimeout verifies that GET /document/{id} returns 504
// when the database layer reports a Cassandra timeout on GetSubDocument.
func TestGetSubDocumentHandlerCassandraTimeout(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.DatabaseClient = &timeoutMockClient{DatabaseClient: server.DatabaseClient}
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	url := fmt.Sprintf("/api/v1/device/%v/document/lan", cpeMac)
	req, err := http.NewRequest("GET", url, nil)
	assert.NilError(t, err)

	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusGatewayTimeout)
}

// TestPostSubDocumentHandlerCassandraTimeout verifies that POST /document/{id} returns 504
// when the database layer reports a Cassandra timeout on GetRootDocumentLabels.
func TestPostSubDocumentHandlerCassandraTimeout(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.DatabaseClient = &timeoutMockClient{DatabaseClient: server.DatabaseClient}
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	url := fmt.Sprintf("/api/v1/device/%v/document/lan", cpeMac)
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte{0x80}))
	assert.NilError(t, err)
	req.Header.Set(common.HeaderContentType, common.HeaderApplicationMsgpack)

	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusGatewayTimeout)
}

// TestMultipartConfigHandlerCassandraTimeout verifies that GET /config returns 504 when
// the database layer reports a Cassandra timeout during document retrieval.
func TestMultipartConfigHandlerCassandraTimeout(t *testing.T) {
	server := NewWebconfigServer(sc, true)
	server.DatabaseClient = &timeoutMockClient{DatabaseClient: server.DatabaseClient}
	router := server.GetRouter(true)

	cpeMac := util.GenerateRandomCpeMac()
	url := fmt.Sprintf("/api/v1/device/%v/config", cpeMac)
	req, err := http.NewRequest("GET", url, nil)
	assert.NilError(t, err)
	req.Header.Set(common.HeaderSchemaVersion, "none")

	res := ExecuteRequest(req, router).Result()
	assert.Equal(t, res.StatusCode, http.StatusGatewayTimeout)
}
