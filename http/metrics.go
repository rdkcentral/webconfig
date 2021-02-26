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
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type AppMetrics struct {
	counter      *prometheus.CounterVec
	duration     *prometheus.HistogramVec
	inFlight     prometheus.Gauge
	responseSize *prometheus.HistogramVec
	requestSize  *prometheus.HistogramVec
}

const AppName = "webconfig"

var (
	urlPatterns = map[string]string{
		`^/api/v1/device/(?P<v0>[^/]+)/document$`: "/api/v1/device/<mac>/document",
		`^/api/v1/device/(?P<v0>[^/]+)/poke$`:     "/api/v1/device/<mac>/poke",
		`^/api/v1/device/(?P<v0>[^/]+)/config$`:   "/api/v1/device/<mac>/config",
	}
)

func GetUrlPattern(url string) string {
	for k, v := range urlPatterns {
		if matched, _ := regexp.MatchString(k, url); matched {
			return v
		}
	}
	return ""
}

func NewMetrics() *AppMetrics {
	metrics := &AppMetrics{
		counter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_requests_total",
				Help: "A counter for total number of requests.",
			},
			[]string{"app", "code", "method", "path"}, // app name, status code, http method, request URL
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_request_duration_seconds",
				Help:    "A histogram of latencies for requests.",
				Buckets: []float64{.01, .02, .05, 0.1, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"app", "code", "method", "path"},
		),
		inFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "in_flight_requests",
				Help: "A gauge of requests currently being served.",
			},
		),
		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_size_bytes",
				Help:    "A histogram of request sizes for requests.",
				Buckets: []float64{200, 500, 1000, 10000, 100000},
			},
			[]string{"app"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "response_size_bytes",
				Help:    "A histogram of response sizes for requests.",
				Buckets: []float64{200, 500, 1000, 10000, 100000},
			},
			[]string{"app"},
		),
	}
	prometheus.MustRegister(metrics.inFlight, metrics.counter, metrics.duration, metrics.responseSize, metrics.requestSize)
	return metrics
}

func WebMetrics(m *AppMetrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promhttp.InstrumentHandlerInFlight(m.inFlight,
			promhttp.InstrumentHandlerDuration(m.duration.MustCurryWith(prometheus.Labels{"app": AppName, "path": GetUrlPattern(r.URL.Path)}),
				promhttp.InstrumentHandlerCounter(m.counter.MustCurryWith(prometheus.Labels{"app": AppName, "path": GetUrlPattern(r.URL.Path)}),
					promhttp.InstrumentHandlerRequestSize(m.requestSize.MustCurryWith(prometheus.Labels{"app": AppName}),
						promhttp.InstrumentHandlerResponseSize(m.responseSize.MustCurryWith(prometheus.Labels{"app": AppName}), next),
					),
				),
			),
		).ServeHTTP(w, r)
	})
}
