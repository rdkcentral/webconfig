package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rdkcentral/webconfig/common"
	log "github.com/sirupsen/logrus"
)

type InternalHttpClient struct {
	*LoopbackClient
}

func NewInternalHttpClient(router *mux.Router) *InternalHttpClient {
	return &InternalHttpClient{
		LoopbackClient: NewLoopbackClient(router),
	}
}

func (c *InternalHttpClient) Do(method, url string, header http.Header, bbytes []byte, fields log.Fields, loggerName string, retry int) (int, []byte, http.Header, bool, error) {
	// verify a response is received
	var req *http.Request
	var err error
	switch method {
	case "GET":
		req, err = http.NewRequest(method, url, nil)
	case "POST", "PATCH":
		req, err = http.NewRequest(method, url, bytes.NewReader(bbytes))
	case "DELETE":
		req, err = http.NewRequest(method, url, nil)
	default:
		return 0, nil, nil, false, common.NewError(fmt.Errorf("method=%v", method))
	}

	if err != nil {
		return 0, nil, nil, false, common.NewError(err)
	}

	if header == nil {
		header = make(http.Header)
	}

	req.Header = header.Clone()
	res, err := c.LoopbackClient.Do(req)
	if err != nil {
		return res.StatusCode, nil, nil, false, common.NewError(err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	rbytes, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, res.Header, false, common.NewError(err)
	}
	return res.StatusCode, rbytes, res.Header, false, nil
}

func (c *InternalHttpClient) DoWithRetries(method, url string, header http.Header, bbytes []byte, fields log.Fields, loggerName string) (int, []byte, http.Header, error) {
	statusCode, rbytes, respHeader, _, err := c.Do(method, url, header, bbytes, fields, loggerName, 0)
	if err != nil {
		return statusCode, rbytes, respHeader, common.NewError(err)
	}
	return statusCode, rbytes, respHeader, nil
}
