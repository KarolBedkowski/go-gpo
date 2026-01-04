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
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
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

	router.Group(func(group chi.Router) {
		group.Use(hlog.RequestIDHandler("req_id", "Request-Id"))

		if cfg.DebugFlags.HasFlag(config.DebugFlightRecorder) {
			group.Use(newFRMiddleware())
		}

		if cfg.DebugFlags.HasFlag(config.DebugTrace) {
			group.Use(newTracingMiddleware(cfg))
		}

		group.Use(logMW)
		group.Use(newRecoverMiddleware)
		group.Use(middleware.CleanPath)
		group.Use(sessionMW)
		group.Use(authMW.handle)
		group.Use(AuthenticatedOnly)
		group.
			With(newPromMiddleware("api", nil)).
			With(middleware.NoCache).
			Mount(cfg.WebRoot+"/", api.Routes())
		group.
			With(newPromMiddleware("web", nil)).
			Mount(cfg.WebRoot+"/web", web.Routes())
		group.
			Get(cfg.WebRoot+"/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, cfg.WebRoot+"/web", http.StatusMovedPermanently)
			})
	})

	if cfg.mgmtEnabledOnMainServer() {
		createMgmtRouters(injector, router, cfg, cfg.WebRoot)
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

func (s *Server) Start(ctx context.Context) error {
	logger := log.Logger

	if s.cfg.DebugFlags.HasFlag(config.DebugRouter) {
		logRoutes(ctx, "Server", s.router)
	}

	listener, err := newListener(ctx, s.cfg.Listen, s.cfg.TLSKey, s.cfg.TLSCert)
	if err != nil {
		return aerr.Wrapf(err, "start listen error")
	}

	logger.Log().Msgf("Server: listen on address=%s https=%v webroot=%q",
		s.cfg.Listen, s.cfg.tlsEnabled(), s.cfg.WebRoot)

	if s.cfg.mgmtEnabledOnMainServer() {
		logger.Warn().Msg("Server: management endpoints enabled on main server")
	}

	go func() {
		if err := s.s.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log().Err(err).Msgf("Server: serve error: %s", err)
		}
	}()

	return nil
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

//-------------------------------------------------------------

func logRoutes(ctx context.Context, name string, r chi.Routes) {
	logger := log.Ctx(ctx)

	walkFunc := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		_ = handler
		_ = middlewares
		route = strings.ReplaceAll(route, "/*/", "/")
		logger.Debug().Msgf("%s: ROUTE: %s %s", name, method, route)

		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		logger.Error().Err(err).Msgf("Server: routers walk error: %s", err)
	}
}

func newListener(ctx context.Context, address, tlskey, tlscert string) (net.Listener, error) {
	if tlskey == "" || tlscert == "" {
		lc := net.ListenConfig{}

		l, err := lc.Listen(ctx, "tcp", address)
		if err != nil {
			return nil, aerr.Wrapf(err, "listen failed").WithMeta("address", address)
		}

		return l, nil
	}

	cert, err := tls.LoadX509KeyPair(tlscert, tlskey)
	if err != nil {
		return nil, aerr.Wrapf(err, "load certificates failed").
			WithMeta("cert", tlscert, "key", tlskey)
	}

	cfg := tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	l, err := tls.Listen("tcp", address, &cfg)
	if err != nil {
		return nil, aerr.Wrapf(err, "tls listen failed").WithMeta("address", address)
	}

	return l, nil
}
