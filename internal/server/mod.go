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
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/service"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

type Configuration struct {
	Listen  string
	LogBody bool
	WebRoot string
}

const (
	connectioTimeout   = 60 * time.Second
	sessionMaxLifetime = 14 * 24 * 60 * 60 // 14d
)

func Start(ctx context.Context, repo *db.Database, cfg *Configuration) error {
	deviceSrv := service.NewDeviceService(repo)
	subSrv := service.NewSubssService(repo)
	usersSrv := service.NewUsersService(repo)
	episodesSrv := service.NewEpisodesService(repo)
	settingsSrv := service.NewSettingsService(repo)
	podcastsSrv := service.NewPodcastsService(repo)

	// middlewares
	sessionMW, err := newSessionMiddleware(repo)
	if err != nil {
		return err
	}

	authMW := authenticator{usersSrv}

	// routes
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat("/ping"))
	router.Use(middleware.CleanPath)
	router.Use(middleware.RealIP)
	router.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	router.Use(newLogMiddleware(cfg))
	router.Use(newRecoverMiddleware)
	router.Use(middleware.Timeout(connectioTimeout))

	router.Method("GET", "/metrics", newMetricsHandler())

	api := gpoapi.New(deviceSrv, subSrv, usersSrv, episodesSrv, settingsSrv)
	router.
		With(newPromMiddleware("api", nil).Handler).
		With(sessionMW).
		With(authMW.handle).
		With(AuthenticatedOnly).
		With(middleware.NoCache).
		Mount("/", api.Routes())

	web := gpoweb.New(deviceSrv, subSrv, usersSrv, episodesSrv, settingsSrv, podcastsSrv, cfg.WebRoot)
	router.
		With(newPromMiddleware("web", nil).Handler).
		With(sessionMW).
		With(authMW.handle).
		With(AuthenticatedOnly).
		Mount("/web", web.Routes())

	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("go-gpo"))
	})

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
