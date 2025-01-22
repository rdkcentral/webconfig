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
package tracing

import (
	"os"
	"strings"

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"

	otelpropagation "go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	AuditIDHeader            = "X-Auditid"
	UserAgentHeader          = "User-Agent"
	DefaultMoracideTagPrefix = "X-Cl-Experiment"
)

type SpanSetterFunc func(log.Fields, string)

// XpcTracer is a wrapper around tracer setup
type XpcTracer struct {
	OtelEnabled       bool
	moracideTagPrefix string // Special request header for moracide expts e.g. canary deployments

	// internal vars used by Otel
	appEnv     string // set this to dev for red, staging for yellow and prod for green
	appName    string
	appVersion string
	appSHA     string // unused
	region     string // AWS Region e.g. us-west-2, unused, use it as a span tagattribute
	siteColor  string // red/yellow/green, unused, use it as a span attribute

	// internal otel vars
	otelEndpoint       string
	otelOpName         string
	otelProvider       string
	otelTracerProvider oteltrace.TracerProvider
	otelPropagator     otelpropagation.TextMapPropagator
	otelTracer         oteltrace.Tracer

	SetSpan SpanSetterFunc
}

func NewXpcTracer(conf *configuration.Config) *XpcTracer {
	xpcTracer := new(XpcTracer)
	initAppData(xpcTracer, conf)
	otelInit(xpcTracer, conf)
	xpcTracer.moracideTagPrefix = conf.GetString("webconfig.tracing.moracide_tag_prefix", DefaultMoracideTagPrefix)

	if xpcTracer.OtelEnabled {
		xpcTracer.SetSpan = OtelSetSpan
	} else {
		xpcTracer.SetSpan = NoopSetSpan
	}

	return xpcTracer
}

func (t *XpcTracer) MoracideTagPrefix() string {
	return t.moracideTagPrefix
}

// otelOpName should return "http.request" by default
func (t *XpcTracer) OtelOpName() string {
	if len(t.otelOpName) == 0 {
		return "http.request"
	}
	return t.otelOpName
}

func (t *XpcTracer) OtelTracerProvider() oteltrace.TracerProvider {
	return t.otelTracerProvider
}

func (t *XpcTracer) AppName() string {
	return t.appName
}

func (t *XpcTracer) AppVersion() string {
	return t.appVersion
}

func (t *XpcTracer) AppEnv() string {
	return t.appEnv
}

func (t *XpcTracer) Region() string {
	return t.region
}

func initAppData(xpcTracer *XpcTracer, conf *configuration.Config) {
	codeGitCommit := strings.Split(conf.GetString("webconfig.code_git_commit"), "-")
	xpcTracer.appName = codeGitCommit[0]
	if len(codeGitCommit) > 1 {
		xpcTracer.appVersion = codeGitCommit[1]
	}
	if len(codeGitCommit) > 2 {
		xpcTracer.appSHA = codeGitCommit[2]
	}

	// Env vars
	xpcTracer.appEnv = "dev"
	siteColor := os.Getenv("site_color")
	if strings.EqualFold(siteColor, "yellow") {
		xpcTracer.appEnv = "staging"
	} else if strings.EqualFold(siteColor, "green") {
		xpcTracer.appEnv = "prod"
	}
	xpcTracer.region = os.Getenv("site_region")
	if xpcTracer.region == "" {
		xpcTracer.region = os.Getenv("site_region_name")
	}
	log.Debugf("site_color = %s, env = %s, region = %s", siteColor, xpcTracer.appEnv, xpcTracer.region)
}

func OtelSetSpan(fields log.Fields, tag string) {
	SetSpanStatusCode(fields)
	SetSpanMoracideTags(fields, tag)
}

func NoopSetSpan(fields log.Fields, tag string) {
}
