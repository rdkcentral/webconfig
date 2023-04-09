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
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-akka/configuration"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promemodel "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
)

type StateCounter struct {
	Deployed        int `json:"deployed"`
	PendingDownload int `json:"pending_download"`
	InDeployment    int `json:"in_deployment"`
	Failure         int `json:"failure"`
}

func (m *StateCounter) String() string {
	bbytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("cannot marshal, err=%v\n", err)
	}
	return string(bbytes)
}

type AppMetrics struct {
	appName                     string
	urlPatternFn                func(string) string
	counter                     *prometheus.CounterVec
	duration                    *prometheus.HistogramVec
	inFlight                    prometheus.Gauge
	responseSize                *prometheus.HistogramVec
	requestSize                 *prometheus.HistogramVec
	stateDeployed               *prometheus.GaugeVec
	statePendingDownload        *prometheus.GaugeVec
	stateInDeployment           *prometheus.GaugeVec
	stateFailure                *prometheus.GaugeVec
	kafkaLag                    *prometheus.SummaryVec
	kafkaDuration               *prometheus.SummaryVec
	eventCounter                *prometheus.CounterVec
	watchedStateDeployed        *prometheus.GaugeVec
	watchedStatePendingDownload *prometheus.GaugeVec
	watchedStateInDeployment    *prometheus.GaugeVec
	watchedStateFailure         *prometheus.GaugeVec
	watchedCpes                 []string
	logrusLevel                 log.Level
}

var (
	urlPatterns = map[string]string{
		`^/api/v1/device/(?P<v0>[^/]+)/document$`: "/api/v1/device/<mac>/document",
		`^/api/v1/device/(?P<v0>[^/]+)/poke$`:     "/api/v1/device/<mac>/poke",
		`^/api/v1/device/(?P<v0>[^/]+)/config$`:   "/api/v1/device/<mac>/config",
	}

	appMetrics *AppMetrics
)

func GetUrlPattern(url string) string {
	if strings.Contains(url, "/config?") || strings.HasSuffix(url, "/config") {
		return "/api/v1/device/<mac>/config"
	}

	for k, v := range urlPatterns {
		if matched, _ := regexp.MatchString(k, url); matched {
			return v
		}
	}
	return ""
}

func NewMetrics(conf *configuration.Config, args ...func(string) string) *AppMetrics {
	if appMetrics != nil {
		return appMetrics
	}

	var fn func(string) string
	if len(args) > 0 {
		fn = args[0]
	} else {
		fn = GetUrlPattern
	}

	// read from the config object
	appName := conf.GetString("webconfig.app_name", "webconfig")
	watchedCpes := conf.GetStringList("webconfig.metrics.watched_cpes")
	metricsLogLevel := conf.GetString("webconfig.metrics.log_level")
	logrusLevel := log.TraceLevel
	if x, err := log.ParseLevel(metricsLogLevel); err == nil {
		logrusLevel = x
	}

	appMetrics = &AppMetrics{
		appName:      appName,
		urlPatternFn: fn,
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
		stateDeployed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_state_deployed",
				Help: "A gauge for the number of cpes in deployed state per feature.",
			},
			[]string{"feature", "client"},
		),
		statePendingDownload: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_state_pending_download",
				Help: "A gauge for the number of cpes in pending_download state per feature.",
			},
			[]string{"feature", "client"},
		),
		stateInDeployment: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_state_in_deployment",
				Help: "A gauge for the number of cpes in in_deployment state per feature.",
			},
			[]string{"feature", "client"},
		),
		stateFailure: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_state_failure",
				Help: "A gauge for the number of cpes in failure state per feature.",
			},
			[]string{"feature", "client"},
		),
		kafkaLag: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: appName + "_kafka_lag",
				Help: "A summary of kafka lag.",
			},
			[]string{"event", "client"},
		),
		kafkaDuration: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: appName + "_kafka_duration",
				Help: "A summary of kafka duration.",
			},
			[]string{"event", "client"},
		),
		eventCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: appName + "_event_types",
				Help: "A counter for kafka event types",
			},
			[]string{"status", "event"}, // app name, kafka processing success/fail, event type (mqtt-get/set, webpa)
		),
		watchedStateDeployed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_watched_state_deployed",
				Help: "A gauge for the number of watched cpes in deployed state per feature.",
			},
			[]string{"feature", "client"},
		),
		watchedStatePendingDownload: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_watched_state_pending_download",
				Help: "A gauge for the number of watched cpes in pending_download state per feature.",
			},
			[]string{"feature", "client"},
		),
		watchedStateInDeployment: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_watched_state_in_deployment",
				Help: "A gauge for the number of watched cpes in in_deployment state per feature.",
			},
			[]string{"feature", "client"},
		),
		watchedStateFailure: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: appName + "_watched_state_failure",
				Help: "A gauge for the number of watched cpes in failure state per feature.",
			},
			[]string{"feature", "client"},
		),
		watchedCpes: watchedCpes,
		logrusLevel: logrusLevel,
	}
	prometheus.MustRegister(
		appMetrics.inFlight,
		appMetrics.counter,
		appMetrics.duration,
		appMetrics.responseSize,
		appMetrics.requestSize,
		appMetrics.stateDeployed,
		appMetrics.statePendingDownload,
		appMetrics.stateInDeployment,
		appMetrics.stateFailure,
		appMetrics.kafkaLag,
		appMetrics.kafkaDuration,
		appMetrics.eventCounter,
		appMetrics.watchedStateDeployed,
		appMetrics.watchedStatePendingDownload,
		appMetrics.watchedStateInDeployment,
		appMetrics.watchedStateFailure,
	)
	return appMetrics
}

func (m *AppMetrics) WatchedCpes() []string {
	return m.watchedCpes
}
func (m *AppMetrics) SetWatchedCpes(watchedCpes []string) {
	m.watchedCpes = watchedCpes
}

func (m *AppMetrics) WebMetrics(next http.Handler) http.Handler {
	GetUrlPattern := m.UrlPatternFn()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promhttp.InstrumentHandlerInFlight(m.inFlight,
			promhttp.InstrumentHandlerDuration(m.duration.MustCurryWith(prometheus.Labels{"app": m.appName, "path": GetUrlPattern(r.URL.Path)}),
				promhttp.InstrumentHandlerCounter(m.counter.MustCurryWith(prometheus.Labels{"app": m.appName, "path": GetUrlPattern(r.URL.Path)}),
					promhttp.InstrumentHandlerRequestSize(m.requestSize.MustCurryWith(prometheus.Labels{"app": m.appName}),
						promhttp.InstrumentHandlerResponseSize(m.responseSize.MustCurryWith(prometheus.Labels{"app": m.appName}), next),
					),
				),
			),
		).ServeHTTP(w, r)
	})
}

func (m *AppMetrics) UrlPatternFn() func(string) string {
	return m.urlPatternFn
}

// deployed(1)
func (m *AppMetrics) DeployedInc(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateDeployed.With(labels).Inc()
	if isWatchedCpe {
		m.watchedStateDeployed.With(labels).Inc()
	}
}

func (m *AppMetrics) DeployedDec(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateDeployed.With(labels).Dec()
	if isWatchedCpe {
		m.watchedStateDeployed.With(labels).Dec()
	}
}

func (m *AppMetrics) DeployedSet(feature string, client string, v float64, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateDeployed.With(labels).Set(v)
	if isWatchedCpe {
		m.watchedStateDeployed.With(labels).Set(v)
	}
}

// pending_download(2)
func (m *AppMetrics) PendingDownloadInc(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.statePendingDownload.With(labels).Inc()
	if isWatchedCpe {
		m.watchedStatePendingDownload.With(labels).Inc()
	}
}

func (m *AppMetrics) PendingDownloadDec(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.statePendingDownload.With(labels).Dec()
	if isWatchedCpe {
		m.watchedStatePendingDownload.With(labels).Dec()
	}
}

func (m *AppMetrics) PendingDownloadSet(feature string, client string, v float64, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.statePendingDownload.With(labels).Set(v)
	if isWatchedCpe {
		m.watchedStatePendingDownload.With(labels).Set(v)
	}
}

// in_deployment(3)
func (m *AppMetrics) InDeploymentInc(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateInDeployment.With(labels).Inc()
	if isWatchedCpe {
		m.watchedStateInDeployment.With(labels).Inc()
	}
}

func (m *AppMetrics) InDeploymentDec(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateInDeployment.With(labels).Dec()
	if isWatchedCpe {
		m.watchedStateInDeployment.With(labels).Dec()
	}
}

func (m *AppMetrics) InDeploymentSet(feature string, client string, v float64, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateInDeployment.With(labels).Set(v)
	if isWatchedCpe {
		m.watchedStateInDeployment.With(labels).Set(v)
	}
}

// failure(4)
func (m *AppMetrics) FailureInc(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateFailure.With(labels).Inc()
	if isWatchedCpe {
		m.watchedStateFailure.With(labels).Inc()
	}
}

func (m *AppMetrics) FailureDec(feature string, client string, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateFailure.With(labels).Dec()
	if isWatchedCpe {
		m.watchedStateFailure.With(labels).Dec()
	}
}

func (m *AppMetrics) FailureSet(feature string, client string, v float64, isWatchedCpe bool) {
	labels := prometheus.Labels{"feature": feature, "client": client}
	m.stateFailure.With(labels).Set(v)
	if isWatchedCpe {
		m.watchedStateFailure.With(labels).Set(v)
	}
}

// Deployed = 1
// PendingDownload = 2
// InDeployment = 3
// Failure = 4

func (m *AppMetrics) UpdateStateMetrics(oldState, newState int, feature, client, cpeMac string, fields log.Fields) {
	var isWatchedCpe bool
	for _, x := range m.watchedCpes {
		if x == cpeMac {
			isWatchedCpe = true
			break
		}
	}

	// decrease the old state gauge
	switch oldState {
	case Deployed:
		// m.DeployedDec(feature, client, isWatchedCpe)
	case PendingDownload:
		m.PendingDownloadDec(feature, client, isWatchedCpe)
	case InDeployment:
		m.InDeploymentDec(feature, client, isWatchedCpe)
	case Failure:
		// m.FailureDec(feature, client, isWatchedCpe)
	}

	// increase the new state gauge
	switch newState {
	case Deployed:
		m.DeployedInc(feature, client, isWatchedCpe)
	case PendingDownload:
		m.PendingDownloadInc(feature, client, isWatchedCpe)
	case InDeployment:
		m.InDeploymentInc(feature, client, isWatchedCpe)
	case Failure:
		m.FailureInc(feature, client, isWatchedCpe)
	}

	// copy the fields for logging
	tfields := make(log.Fields)
	for k, v := range fields {
		tfields[k] = v
	}

	nfields := log.Fields{
		"logger":         "metrics",
		"old_state":      oldState,
		"new_state":      newState,
		"subdoc_id":      feature,
		"metrics_agent":  client,
		"is_watched_cpe": isWatchedCpe,
	}
	for k, v := range nfields {
		tfields[k] = v
	}

	sfields := m.GetStateCountsAsFields(feature, client)
	for k, v := range sfields {
		tfields[k] = v
	}

	log.WithFields(tfields).Log(m.logrusLevel, "OK")
}

func (m *AppMetrics) ObserveKafkaLag(eventName string, clientName string, lag int) {
	labels := prometheus.Labels{"event": eventName, "client": clientName}
	m.kafkaLag.With(labels).Observe(float64(lag))
}

func (m *AppMetrics) ObserveKafkaDuration(eventName string, clientName string, duration int) {
	labels := prometheus.Labels{"event": eventName, "client": clientName}
	m.kafkaDuration.With(labels).Observe(float64(duration))
}

func (m *AppMetrics) CountKafkaEvents(eventName string, status string) {
	labels := prometheus.Labels{"event": eventName, "status": status}
	m.eventCounter.With(labels).Inc()
}

func (m *AppMetrics) GetStateCounter(feature, client string) (*StateCounter, error) {
	// REMINDER if a label is defined with 2 dimensions, then it must be referred
	//          with 2 dimensions. Aggregation happens at prometheus level
	labels := prometheus.Labels{"feature": feature, "client": client}

	var err error
	var sc StateCounter
	var ptr *int

	for i := 1; i < 5; i++ {
		pm := &promemodel.Metric{}

		switch i {
		case Deployed:
			err = m.stateDeployed.With(labels).Write(pm)
			ptr = &sc.Deployed
		case PendingDownload:
			err = m.statePendingDownload.With(labels).Write(pm)
			ptr = &sc.PendingDownload
		case InDeployment:
			err = m.stateInDeployment.With(labels).Write(pm)
			ptr = &sc.InDeployment
		case Failure:
			err = m.stateFailure.With(labels).Write(pm)
			ptr = &sc.Failure
		}

		if err != nil {
			return nil, NewError(err)
		}
		*ptr = int(pm.Gauge.GetValue())
	}
	return &sc, nil
}

func (m *AppMetrics) GetStateCountsAsFields(feature, client string) log.Fields {
	labels := prometheus.Labels{"feature": feature, "client": client}
	sfields := make(log.Fields)

	pm := &promemodel.Metric{}
	if err := m.stateDeployed.With(labels).Write(pm); err == nil {
		sfields["state_deployed_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.statePendingDownload.With(labels).Write(pm); err == nil {
		sfields["state_pending_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.stateInDeployment.With(labels).Write(pm); err == nil {
		sfields["state_indeployment_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.stateFailure.With(labels).Write(pm); err == nil {
		sfields["state_failuer_count"] = int(pm.Gauge.GetValue())
	}

	// watched list
	pm = &promemodel.Metric{}
	if err := m.watchedStateDeployed.With(labels).Write(pm); err == nil {
		sfields["watched_state_deployed_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.watchedStatePendingDownload.With(labels).Write(pm); err == nil {
		sfields["watched_state_pending_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.watchedStateInDeployment.With(labels).Write(pm); err == nil {
		sfields["watched_state_indeployment_count"] = int(pm.Gauge.GetValue())
	}
	pm = &promemodel.Metric{}
	if err := m.watchedStateFailure.With(labels).Write(pm); err == nil {
		sfields["watched_state_failuer_count"] = int(pm.Gauge.GetValue())
	}

	return sfields
}

func (m *AppMetrics) ResetStateGauges() {
	m.stateDeployed.Reset()
	m.statePendingDownload.Reset()
	m.stateInDeployment.Reset()
	m.stateFailure.Reset()
}

type SimpleMetrics struct {
	Deployed        map[string]int `json:"deployed"`
	PendingDownload map[string]int `json:"pending_download"`
	InDeployment    map[string]int `json:"in_deployment"`
	Failure         map[string]int `json:"failure"`
	KafkaLag        map[string]int `json:"kafka_lag"`
	KafkaDuration   map[string]int `json:"kafka_duration"`
}

func (m *SimpleMetrics) String() string {
	bbytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("cannot marshal, err=%v\n", err)
	}
	return string(bbytes)
}

func ParseGauge(metrics []*promemodel.Metric) map[string]int {
	gaugeMap := make(map[string]int)
	for _, m := range metrics {
		labelPairs := m.GetLabel()
		var feature, client, gaugeKey string
		for _, labelPair := range labelPairs {
			labelName := labelPair.GetName()
			switch labelName {
			case "feature":
				feature = labelPair.GetValue()
			case "client":
				client = labelPair.GetValue()
			}
		}
		if client == "default" {
			gaugeKey = feature
		} else {
			gaugeKey = fmt.Sprintf("%v_%v", feature, client)
		}
		gaugeMap[gaugeKey] = int(m.GetGauge().GetValue())
	}
	return gaugeMap
}

func ParseSummary(metrics []*promemodel.Metric) map[string]int {
	syMap := map[string]int{}

	for _, m := range metrics {
		labelPairs := m.GetLabel()
		var event, client, syKey string
		for _, labelPair := range labelPairs {
			labelName := labelPair.GetName()
			switch labelName {
			case "event":
				event = labelPair.GetValue()
			case "client":
				client = labelPair.GetValue()
			}
		}
		if client == "default" {
			syKey = event
		} else {
			syKey = fmt.Sprintf("%v_%v", event, client)
		}
		syMap[syKey] = int(m.GetSummary().GetSampleSum())
	}
	return syMap
}
