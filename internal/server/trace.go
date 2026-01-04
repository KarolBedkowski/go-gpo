//go:build trace

package server

//
// tracing.go
// Copyright (C) 2026 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/config"
	xtrace "golang.org/x/net/trace"
)

func newTracingMiddleware(cfg *config.ServerConf) func(http.Handler) http.Handler {
	xtrace.AuthRequest = cfg.AuthMgmtRequest

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// skip tracing for the request to statics etc.
			if shouldSkipLogRequest(request) {
				next.ServeHTTP(writer, request)

				return
			}

			ctx := request.Context()
			reqid := "?"

			if id, ok := hlog.IDFromCtx(ctx); ok {
				reqid = id.String()
				pprof.SetGoroutineLabels(pprof.WithLabels(ctx, pprof.Labels("reqid", reqid)))
			}

			tr := xtrace.New("server", request.URL.Path+" req_id="+reqid)
			defer tr.Finish()

			ctx = xtrace.NewContext(ctx, tr)
			request = request.WithContext(ctx)

			next.ServeHTTP(writer, request)
		})
	}
}

func mountXTrace(group chi.Router, webroot string) {
	group.Get(webroot+"/debug/requests", xtrace.Traces)
	group.Get(webroot+"/debug/events", xtrace.Events)
}

//-------------------------------------------------------------

const FlightRecorderThreshold = 200 * time.Millisecond

type frMiddleware struct {
	once sync.Once
	fr   *trace.FlightRecorder
}

func newFRMiddleware() func(http.Handler) http.Handler {
	frm := &frMiddleware{}

	frm.fr = trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   FlightRecorderThreshold,
		MaxBytes: 1 << 20, //nolint:mnd  // 1MB
	})

	if err := frm.fr.Start(); err != nil {
		log.Logger.Error().Err(err).Msgf("FlightRecorder: start error=%q", err)

		frm.once.Do(func() {})

		return func(next http.Handler) http.Handler {
			return next
		}
	}

	log.Logger.Warn().Msgf("FlightRecorder: enabled; threshold=%s", FlightRecorderThreshold)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			if frm.fr.Enabled() && time.Since(start) > FlightRecorderThreshold {
				go frm.captureSnapshot(r.Context())
			}
		})
	}
}

func (f *frMiddleware) captureSnapshot(ctx context.Context) {
	// once.Do ensures that the provided function is executed only once.
	f.once.Do(func() {
		logger := log.Logger
		fname := time.Now().Format(time.RFC3339)

		reqid := "unk"
		if id, ok := hlog.IDFromCtx(ctx); ok {
			reqid = id.String()
		}

		fout, err := os.Create("snapshot" + fname + reqid + ".trace")
		if err != nil {
			logger.Error().Err(err).Msgf("FlightRecorder: opening snapshot file %q error=%q", fout.Name(), err)

			return
		}
		defer fout.Close()

		// WriteTo writes the flight recorder data to the provided io.Writer.
		if _, err = f.fr.WriteTo(fout); err != nil {
			logger.Error().Err(err).Msgf("FlightRecorder: writing snapshot to file %q error=%q", fout.Name(), err)

			return
		}

		// Stop the flight recorder after the snapshot has been taken.
		f.fr.Stop()
		logger.Warn().Str(common.LogKeyReqID, reqid).
			Msgf("FlightRecorder: captured snapshot to %q for req_id=%s", fout.Name(), reqid)
	})
}
