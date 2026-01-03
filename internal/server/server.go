//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	dochi "github.com/samber/do/http/chi/v2"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/config"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

const (
	sessionMaxLifetime    = (15 * 60) * time.Second //nolint:mnd
	defaultReadTimeout    = 60 * time.Second
	defaultWriteTimeout   = 60 * time.Second
	defaultMaxHeaderBytes = 1 << 20
	shutdownTimeout       = 10 * time.Second
)

type Server struct {
	router chi.Router

	cfg *Configuration
	s   *http.Server
}

func New(injector do.Injector) (*Server, error) {
	cfg := do.MustInvoke[*Configuration](injector)
	authMW := do.MustInvoke[authenticator](injector)
	api := do.MustInvoke[gpoapi.API](injector)
	web := do.MustInvoke[gpoweb.WEB](injector)
	sessionMW := do.MustInvoke[sessionMiddleware](injector)
	logMW := do.MustInvoke[logMiddleware](injector)

	// routes
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat(cfg.WebRoot + "/ping"))
	router.Use(middleware.RealIP)
	router.Get(cfg.WebRoot+"/health", newHealthChecker(injector))

	router.Group(func(group chi.Router) {
		group.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
		group.Use(logMW)
		group.Use(newRecoverMiddleware)
		group.Use(middleware.CleanPath)
		group.Use(sessionMW)
		group.Use(authMW.handle)
		group.Use(AuthenticatedOnly)
		group.
			With(newPromMiddleware("api", nil).Handler).
			With(middleware.NoCache).
			Mount(cfg.WebRoot+"/", api.Routes())
		group.
			With(newPromMiddleware("web", nil).Handler).
			Mount(cfg.WebRoot+"/web", web.Routes())
		group.
			Get(cfg.WebRoot+"/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, cfg.WebRoot+"/web", http.StatusMovedPermanently)
			})
	})

	if cfg.DebugFlags.HasFlag(config.DebugDo) {
		dochi.Use(router, cfg.WebRoot+"/debug/do", injector)
	}

	if cfg.DebugFlags.HasFlag(config.DebugGo) {
		router.Mount(cfg.WebRoot+"/debug", middleware.Profiler())
	}

	if cfg.EnableMetrics {
		router.Method("GET", cfg.WebRoot+"/metrics", newMetricsHandler())
	}

	return &Server{
		router: router,
		cfg:    cfg,
		s: &http.Server{
			Addr:           cfg.Listen,
			Handler:        router,
			ReadTimeout:    defaultReadTimeout,
			WriteTimeout:   defaultWriteTimeout,
			MaxHeaderBytes: defaultMaxHeaderBytes,
		},
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	if s.cfg.DebugFlags.HasFlag(config.DebugRouter) {
		logRoutes(ctx, s.router)
	}

	if err := s.s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen error: %w", err)
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.Logger

	if s.cfg.DebugFlags.HasFlag(config.DebugRouter) {
		logRoutes(ctx, s.router)
	}

	listener, err := s.newListener(ctx)
	if err != nil {
		return aerr.Wrapf(err, "start listen error")
	}

	logger.Log().Msgf("Server: listen on address=%s https=%v", s.cfg.Listen, s.cfg.tlsEnabled())

	go func() {
		if err := s.s.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log().Err(err).Msgf("Server: serve error: %s", err)
		}
	}()

	return nil
}

func (s *Server) Stop(_ error) {
	logger := log.Logger

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.s.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msgf("Server: shutdown server failed: %s", err)
	} else {
		logger.Debug().Msg("Server: stopped")
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("Server: stopping...")

	if err := s.s.Shutdown(ctx); err != nil {
		return aerr.Wrapf(err, "shutdown server failed")
	}

	logger.Debug().Msg("Server: stopped")

	return nil
}

func logRoutes(ctx context.Context, r chi.Routes) {
	logger := log.Ctx(ctx)

	walkFunc := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		_ = handler
		_ = middlewares
		route = strings.ReplaceAll(route, "/*/", "/")
		logger.Debug().Msgf("Server: ROUTE: %s %s", method, route)

		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		logger.Error().Err(err).Msgf("Server: routers walk error: %s", err)
	}
}

func (s *Server) newListener(ctx context.Context) (net.Listener, error) {
	if s.cfg.TLSKey == "" || s.cfg.TLSCert == "" {
		lc := net.ListenConfig{}

		l, err := lc.Listen(ctx, "tcp", s.cfg.Listen)
		if err != nil {
			return nil, aerr.Wrapf(err, "listen failed").WithMeta("address", s.cfg.Listen)
		}

		return l, nil
	}

	cert, err := tls.LoadX509KeyPair(s.cfg.TLSCert, s.cfg.TLSKey)
	if err != nil {
		return nil, aerr.Wrapf(err, "load certificates failed").
			WithMeta("cert", s.cfg.TLSCert, "key", s.cfg.TLSKey)
	}

	cfg := tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	l, err := tls.Listen("tcp", s.cfg.Listen, &cfg)
	if err != nil {
		return nil, aerr.Wrapf(err, "tls listen failed").WithMeta("address", s.cfg.Listen)
	}

	return l, nil
}

// newHealthChecker create new handler for /health endpoint. Accept only connection from localhost.
func newHealthChecker(injector do.Injector) http.HandlerFunc {
	rootscope := injector.RootScope()

	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.RemoteAddr, "localhost") && !strings.HasPrefix(r.RemoteAddr, "127.0.0.1") {
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
