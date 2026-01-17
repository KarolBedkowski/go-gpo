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
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	dochi "github.com/samber/do/http/chi/v2"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type MgmtServer struct {
	router chi.Router

	cfg *config.ServerConf
	s   *http.Server
}

func NewMgmt(injector do.Injector) (*MgmtServer, error) {
	cfg := do.MustInvoke[*config.ServerConf](injector)

	// routes
	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Heartbeat(cfg.MgmtServer.WebRoot + "/ping"))

	createMgmtRouters(injector, router, cfg, cfg.MgmtServer)

	return &MgmtServer{
		router: router,
		cfg:    cfg,
		s: &http.Server{
			Addr:           cfg.MgmtServer.Address,
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

	scfg := s.cfg.MgmtServer

	listener, err := newListener(ctx, scfg)
	if err != nil {
		return aerr.Wrapf(err, "start listen error")
	}

	logger.Log().
		Msgf("MgmtServer: listen on address=%s https=%v webroot=%q", scfg.Address, scfg.TLSEnabled(), scfg.WebRoot)

	go func() {
		if err := s.s.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log().Err(err).Msgf("Server: serve error: %s", err)
		}
	}()

	return nil
}

func (s *MgmtServer) Shutdown(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("MgmtServer: stopping...")

	if err := s.s.Shutdown(ctx); err != nil {
		return aerr.Wrapf(err, "shutdown server failed")
	}

	logger.Debug().Msg("MgmtServer: stopped")

	return nil
}

//-------------------------------------------------------------

func createMgmtRouters(injector do.Injector, router *chi.Mux, cfg *config.ServerConf, scfg config.ListenConf) {
	webroot := scfg.WebRoot

	router.Get(webroot+"/health", newHealthChecker(injector, cfg))

	router.Group(func(group chi.Router) {
		group.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
		group.Use(newVerySimpleLogMiddleware("MgmtServer"))
		group.Use(newRecoverMiddleware)
		group.Use(middleware.CleanPath)
		group.Use(newAuthMgmtMiddleware(cfg))

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
func newHealthChecker(injector do.Injector, cfg *config.ServerConf) http.HandlerFunc {
	rootscope := injector.RootScope()

	return func(w http.ResponseWriter, r *http.Request) {
		log.Logger.Debug().Msgf("remote %v", r.RemoteAddr)

		// access to /health only from selected networks
		if _, access := cfg.AuthMgmtRequest(r); !access {
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
