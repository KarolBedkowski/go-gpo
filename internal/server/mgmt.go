//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	dochi "github.com/samber/do/http/chi/v2"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type MgmtServer struct {
	router chi.Router

	cfg *Configuration
	s   *http.Server
}

func NewMgmt(injector do.Injector) (*MgmtServer, error) {
	cfg := do.MustInvoke[*Configuration](injector)

	// routes
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat(cfg.MgmtWebRoot + "/ping"))
	router.Use(middleware.RealIP)

	createMgmtRouters(injector, router, cfg, cfg.MgmtWebRoot)

	return &MgmtServer{
		router: router,
		cfg:    cfg,
		s: &http.Server{
			Addr:           cfg.MgmtListen,
			Handler:        router,
			ReadTimeout:    defaultReadTimeout,
			WriteTimeout:   defaultWriteTimeout,
			MaxHeaderBytes: defaultMaxHeaderBytes,
		},
	}, nil
}

func (s *MgmtServer) Start(ctx context.Context) error {
	logger := log.Logger

	if s.cfg.DebugFlags.HasFlag(config.DebugRouter) {
		logRoutes(ctx, "MgmtServer", s.router)
	}

	listener, err := newListener(ctx, s.cfg.MgmtListen, s.cfg.MgmtTLSKey, s.cfg.MgmtTLSCert)
	if err != nil {
		return aerr.Wrapf(err, "start listen error")
	}

	logger.Log().Msgf("MgmtServer: listen on address=%s https=%v webroot=%q",
		s.cfg.MgmtListen, s.cfg.mgmtTLSEnabled(), s.cfg.MgmtWebRoot)

	go func() {
		if err := s.s.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log().Err(err).Msgf("Server: serve error: %s", err)
		}
	}()

	return nil
}

func (s *MgmtServer) Shutdown(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("Server: stopping...")

	if err := s.s.Shutdown(ctx); err != nil {
		return aerr.Wrapf(err, "shutdown server failed")
	}

	logger.Debug().Msg("Server: stopped")

	return nil
}

//-------------------------------------------------------------

func createMgmtRouters(injector do.Injector, router *chi.Mux, cfg *Configuration, webroot string) {
	router.Get(cfg.MgmtWebRoot+"/health", newHealthChecker(injector, cfg))

	router.Group(func(group chi.Router) {
		group.Use(newAuthDebugMiddleware(cfg))

		if cfg.DebugFlags.HasFlag(config.DebugDo) {
			dochi.Use(router, webroot+"/debug/do", injector)
		}

		if cfg.DebugFlags.HasFlag(config.DebugGo) {
			group.Mount(webroot+"/debug", middleware.Profiler())
		}

		if cfg.DebugFlags.HasFlag(config.DebugTrace) {
			mountXTrace(group, webroot)
		}
	})

	if cfg.EnableMetrics {
		router.Method("GET", webroot+"/metrics", newMetricsHandler())
	}
}

//-------------------------------------------------------------

// newHealthChecker create new handler for /health endpoint. Accept only connection from localhost.
func newHealthChecker(injector do.Injector, cfg *Configuration) http.HandlerFunc {
	rootscope := injector.RootScope()

	return func(w http.ResponseWriter, r *http.Request) {
		log.Logger.Debug().Msgf("remote %v", r.RemoteAddr)

		// access to /health only from localhost
		if _, access := cfg.authDebugRequest(r); !access {
			w.WriteHeader(http.StatusForbidden)

			return
		}

		response := "ok"

		for service, err := range rootscope.HealthCheckWithContext(r.Context()) {
			if err != nil {
				log.Logger.Error().Err(err).Str("service", service).
					Msgf("HealthChecker: service=%q failed on healthcheck: %s", service, err)

				response = "error"
			}
		}

		render.PlainText(w, r, response)
	}
}
