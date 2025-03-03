package http

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type APIClient interface {
	Do(string, string, http.Header, []byte, log.Fields, string, int) (int, []byte, http.Header, bool, error)
	DoWithRetries(string, string, http.Header, []byte, log.Fields, string) (int, []byte, http.Header, error)
}
