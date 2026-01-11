package server

//
// instrumentation.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type promMiddleware struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.SummaryVec
	responseSize    *prometheus.SummaryVec
	inFlight        prometheus.Gauge
}

func (m *promMiddleware) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := promhttp.InstrumentHandlerInFlight(m.inFlight, next)
		base = promhttp.InstrumentHandlerResponseSize(m.responseSize, base)
		base = promhttp.InstrumentHandlerRequestSize(m.requestSize, base)
		base = promhttp.InstrumentHandlerDuration(m.requestDuration, base)
		base = promhttp.InstrumentHandlerCounter(m.requestsTotal, base)

		base.ServeHTTP(w, r)
	})
}

// New returns a Middleware interface.
func newPromMiddleware(name string, buckets []float64) func(http.Handler) http.Handler {
	if buckets == nil {
		buckets = []float64{0.05, 0.1, 0.5, 1, 2, 5}
	}

	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": name}, prometheus.DefaultRegisterer)

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
			Buckets: buckets,
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
	inFlight := promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	mw := promMiddleware{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		inFlight:        inFlight,
	}

	return mw.handler
}

func newMetricsHandler() http.Handler {
	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{DisableCompression: true}),
	)
}
