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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	dochi "github.com/samber/do/http/chi/v2"
	"github.com/samber/do/v2"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/config"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

type Configuration struct {
	Listen     string
	WebRoot    string
	DebugFlags config.DebugFlags
}

const (
	sessionMaxLifetime    = 14 * 24 * 60 * 60 // 14d
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

func New(injector do.Injector) (Server, error) {
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

	router.Method("GET", cfg.WebRoot+"/metrics", newMetricsHandler())

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

	return Server{
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
	if s.cfg.DebugFlags.HasFlag(config.DebugRouter) {
		logRoutes(ctx, s.router)
	}

	if err := s.s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen error: %w", err)
	}

	return nil
}

func (s *Server) Stop(_ error) {
	logger := log.Logger.With().Str("mod", "server").Logger()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.s.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("shutdown server failed")
	} else {
		logger.Debug().Msg("server stopped")
	}
}

func logRoutes(ctx context.Context, r chi.Routes) {
	logger := log.Ctx(ctx).With().Str("mod", "server").Logger()

	walkFunc := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		_ = handler
		_ = middlewares
		route = strings.ReplaceAll(route, "/*/", "/")
		logger.Debug().Msgf("ROUTE: %s %s", method, route)

		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		logger.Error().Err(err).Msg("routers walk error")
	}
}
