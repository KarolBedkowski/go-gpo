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

	cfg *config.ServerConf
	s   *http.Server
}

func New(injector do.Injector) (*Server, error) {
	cfg := do.MustInvoke[*config.ServerConf](injector)
	authMW := do.MustInvoke[authenticator](injector)
	api := do.MustInvoke[gpoapi.API](injector)
	web := do.MustInvoke[gpoweb.WEB](injector)
	sessionMW := do.MustInvoke[sessionMiddleware](injector)
	logMW := do.MustInvoke[logMiddleware](injector)
	webroot := cfg.MainServer.WebRoot

	// routes
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat(webroot + "/ping"))
	router.Use(middleware.RealIP)

	if cfg.SetSecurityHeaders {
		router.Use(SecHeadersMiddleware)
	}

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
			Mount(webroot+"/", api.Routes())
		group.
			With(newPromMiddleware("web", nil)).
			Mount(webroot+"/web", web.Routes())
		group.Get(webroot+"/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, webroot+"/web", http.StatusMovedPermanently)
		})
	})

	if cfg.MgmtEnabledOnMainServer() {
		createMgmtRouters(injector, router, cfg, cfg.MainServer)
	}

	return &Server{
		router: router,
		cfg:    cfg,
		s: &http.Server{
			Addr:           cfg.MainServer.Address,
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

	scfg := s.cfg.MainServer

	listener, err := newListener(ctx, scfg)
	if err != nil {
		return aerr.Wrapf(err, "start listen error")
	}

	logger.Log().Msgf("Server: listen on address=%s https=%v webroot=%q",
		scfg.Address, scfg.TLSEnabled(), scfg.WebRoot)

	if s.cfg.MgmtEnabledOnMainServer() {
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

func newListener(ctx context.Context, scfg config.ListenConf) (net.Listener, error) {
	if scfg.TLSKey == "" || scfg.TLSCert == "" {
		lc := net.ListenConfig{}

		l, err := lc.Listen(ctx, "tcp", scfg.Address)
		if err != nil {
			return nil, aerr.Wrapf(err, "listen failed").WithMeta("address", scfg.Address)
		}

		return l, nil
	}

	cert, err := tls.LoadX509KeyPair(scfg.TLSCert, scfg.TLSKey)
	if err != nil {
		return nil, aerr.Wrapf(err, "load certificates failed").
			WithMeta("cert", scfg.TLSCert, "key", scfg.TLSKey)
	}

	cfg := tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	l, err := tls.Listen("tcp", scfg.Address, &cfg)
	if err != nil {
		return nil, aerr.Wrapf(err, "tls listen failed").WithMeta("address", scfg.Address)
	}

	return l, nil
}
