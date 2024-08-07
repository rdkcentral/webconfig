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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
)

// Tracing contains the core dependencies to make tracing possible across an application.
type otelTracing struct {
	providerName   string
	envName        string
	appName        string
	opName         string
	tracerProvider trace.TracerProvider
	propagator     propagation.TextMapPropagator
	tracer         trace.Tracer
}

type providerConstructor func(conf *configuration.Config) (trace.TracerProvider, error)

var (
	ErrTracerProviderNotFound    = errors.New("TracerProvider builder could not be found")
	ErrTracerProviderBuildFailed = errors.New("Failed building TracerProvider")
	providersConfig              = map[string]providerConstructor{
		"http":   httpTraceProvider,
		"stdout": stdoutTraceProvider,
		"noop":   noopTraceProvider,
	}

	otelTracer otelTracing
)

// DefaultTracerProvider is used when no provider is given.
// The Noop tracer provider turns all tracing related operations into
// noops essentially disabling tracing.
const defaultTracerProvider = "noop"

// newOtel creates a structure with components that apps can use to initialize OpenTelemetry
// tracing instrumentation code.
func newOtel(conf *configuration.Config) (*otelTracing, error) {
	if IsNoopTracing(conf) {
		log.Debug("open telemetry tracing disabled (noop)")
	} else {
		log.Debug("opentelemetry tracing enabled")
	}

	otelTracer.appName = conf.GetString("webconfig.app_name")
	otelTracer.providerName = conf.GetString("webconfig.opentelemetry.provider", defaultTracerProvider)
	otelTracer.envName = conf.GetString("webconfig.opentelemetry.env_name", "dev")
	otelTracer.opName = conf.GetString("webconfig.opentelemetry.operation_name", "http.request")
	tracerProvider, err := newTracerProvider(conf)
	if err != nil {
		return &otelTracer, err
	}
	otelTracer.tracerProvider = tracerProvider
	otel.SetTracerProvider(tracerProvider)

	// Set up propagator.
	prop := newPropagator()
	otelTracer.propagator = prop
	otel.SetTextMapPropagator(prop)

	otelTracer.tracer = otel.Tracer(otelTracer.appName)
	return &otelTracer, nil
}

// IsNoopTracing returns true if the provider is set to "noop"
func IsNoopTracing(conf *configuration.Config) bool {
	providerName := conf.GetString("webconfig.opentelemetry.provider", defaultTracerProvider)
	return strings.EqualFold(providerName, "noop")
}

// TracerProvider returns the tracer provider component. By default, the noop
// tracer provider is returned.
func (t otelTracing) TracerProvider() trace.TracerProvider {
	if t.tracerProvider == nil {
		return noop.NewTracerProvider()
	}
	return t.tracerProvider
}

// Propagator returns the component that helps propagate trace context across
// API boundaries. By default, a W3C Trace Context format propagator is returned.
func (t otelTracing) Propagator() propagation.TextMapPropagator {
	if t.propagator == nil {
		return propagation.TraceContext{}
	}
	return t.propagator
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTracerProvider creates the TracerProvider based on config setting
// If no config setting, a noop tracerProvider will be returned.
func newTracerProvider(conf *configuration.Config) (trace.TracerProvider, error) {
	providerName := conf.GetString("webconfig.opentelemetry.provider", defaultTracerProvider)
	if len(providerName) == 0 {
		providerName = defaultTracerProvider
	}
	// Handling camelcase of provider.
	providerName = strings.ToLower(providerName)
	providerConfig := providersConfig[providerName]
	if providerConfig == nil {
		return nil, fmt.Errorf("%w for provider %s", ErrTracerProviderNotFound, providerName)
	}

	traceProvider, err := providerConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTracerProviderBuildFailed, err)
	}
	return traceProvider, nil
}

func noopTraceProvider(conf *configuration.Config) (trace.TracerProvider, error) {
	return noop.NewTracerProvider(), nil
}

func stdoutTraceProvider(conf *configuration.Config) (trace.TracerProvider, error) {
	option := stdouttrace.WithPrettyPrint()
	exporter, err := stdouttrace.New(option)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter),
		sdktrace.WithBatcher(exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(otelTracer.appName),
				semconv.ServiceNamespaceKey.String(otelTracer.envName),
			),
		),
	)
	return tp, nil
}

func httpTraceProvider(conf *configuration.Config) (trace.TracerProvider, error) {
	// Send traces over HTTP
	endpoint := conf.GetString("webconfig.opentelemetry.endpoint")
	if endpoint == "" {
		return nil, ErrTracerProviderBuildFailed
	}
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTracerProviderBuildFailed, err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(otelTracer.appName),
				semconv.ServiceNamespaceKey.String(otelTracer.envName),
			),
		),
	), nil
}

func (s *WebconfigServer) OtelShutdown() {
	sdkTraceProvider, ok := s.otelTracer.tracerProvider.(*sdktrace.TracerProvider)
	if ok && sdkTraceProvider != nil {
		sdkTraceProvider.Shutdown(context.TODO())
	}
}
