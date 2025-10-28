package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//
// instrumentation.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type promMiddleware struct {
	name    string
	buckets []float64
}

func (m *promMiddleware) Handler(handler http.Handler) http.Handler {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": m.name}, prometheus.DefaultRegisterer)

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code"},
	)
	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: m.buckets,
		},
		[]string{"method", "code"},
	)
	requestSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		[]string{"method", "code"},
	)
	responseSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Tracks the size of HTTP responses.",
		},
		[]string{"method", "code"},
	)
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	base := promhttp.InstrumentHandlerCounter(
		requestsTotal,
		promhttp.InstrumentHandlerDuration(
			requestDuration,
			promhttp.InstrumentHandlerRequestSize(
				requestSize,
				promhttp.InstrumentHandlerResponseSize(
					responseSize,
					promhttp.InstrumentHandlerInFlight(
						inFlight,
						handler,
					),
				),
			),
		),
	)

	return http.HandlerFunc(base.ServeHTTP)
}

// New returns a Middleware interface.
func newPromMiddleware(name string, buckets []float64) *promMiddleware {
	if buckets == nil {
		buckets = []float64{0.5, 2, 10, 30, 90}
	}

	return &promMiddleware{
		name:    name,
		buckets: buckets,
	}
}

func newMetricsHandler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{DisableCompression: true}),
	)
}
