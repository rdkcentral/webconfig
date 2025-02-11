package http

import (
	"bytes"
	"io"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
)

type LoopbackResponseWriter struct {
	header http.Header
	bbytes []byte
	status int
}

func NewLoopbackResponseWriter() *LoopbackResponseWriter {
	return &LoopbackResponseWriter{
		make(http.Header),
		nil,
		0,
	}
}

func (w *LoopbackResponseWriter) Header() http.Header {
	return w.header
}

func (w *LoopbackResponseWriter) Write(bbytes []byte) (int, error) {
	w.bbytes = slices.Clone(bbytes)
	return len(bbytes), nil
}

func (w *LoopbackResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *LoopbackResponseWriter) Body() []byte {
	return w.bbytes
}

func (w *LoopbackResponseWriter) Status() int {
	return w.status
}

// -------------------------

type LoopbackClient struct {
	router *mux.Router
}

func NewLoopbackClient(router *mux.Router) *LoopbackClient {
	return &LoopbackClient{
		router: router,
	}
}

func (c *LoopbackClient) Do(req *http.Request) (*http.Response, error) {
	rw := NewLoopbackResponseWriter()
	c.router.ServeHTTP(rw, req)
	// fmt.Printf("req = %v\n", req)
	// fmt.Printf("status = %v\n", rw.Status())
	// fmt.Printf("body = %s\n", rw.Body())

	body := rw.Body()
	// reader := bytes.NewReader(body)
	// Use io.NopCloser to wrap the reader
	reader := io.NopCloser(bytes.NewReader(body))

	// Create a new http.Response
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body:          reader,           // Set the response body
		ContentLength: int64(len(body)), // Set the content length
	}
	return resp, nil
}
