//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package server

import (
	"context"
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
	connectioTimeout   = 60 * time.Second
	sessionMaxLifetime = 14 * 24 * 60 * 60 // 14d
)

func Start(ctx context.Context, injector do.Injector, cfg *Configuration) error {
	// middlewares
	sessionMW, err := newSessionMiddleware(injector)
	if err != nil {
		return err
	}

	authMW := do.MustInvoke[authenticator](injector)

	// routes
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat(cfg.WebRoot + "/ping"))
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(connectioTimeout))

	router.Method("GET", cfg.WebRoot+"/metrics", newMetricsHandler())

	router.Group(func(group chi.Router) {
		group.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
		group.Use(newLogMiddleware(cfg))
		group.Use(newRecoverMiddleware)
		group.Use(middleware.CleanPath)
		group.Use(sessionMW)
		group.Use(authMW.handle)
		group.Use(AuthenticatedOnly)

		api := do.MustInvoke[gpoapi.API](injector)
		group.
			With(newPromMiddleware("api", nil).Handler).
			With(middleware.NoCache).
			Mount(cfg.WebRoot+"/", api.Routes())

		web := do.MustInvoke[gpoweb.WEB](injector)
		group.
			With(newPromMiddleware("web", nil).Handler).
			Mount(cfg.WebRoot+"/web", web.Routes())

		group.Get(cfg.WebRoot+"/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, cfg.WebRoot+"/web", http.StatusMovedPermanently)
		})
	})

	if cfg.DebugFlags.HasFlag(config.DebugDo) {
		dochi.Use(router, cfg.WebRoot+"/debug/do", injector)
	}

	if cfg.DebugFlags.HasFlag(config.DebugGo) {
		router.Mount(cfg.WebRoot+"/debug", middleware.Profiler())
	}

	logRoutes(ctx, router)

	if err := http.ListenAndServe(cfg.Listen, router); err != nil { //nolint:gosec
		return fmt.Errorf("start listen error: %w", err)
	}

	return nil
}

func logRoutes(ctx context.Context, r chi.Routes) {
	logger := log.Ctx(ctx)
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
