//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type Configuration struct {
	NoAuth  bool
	Listen  string
	LogBody bool
}

const (
	connectioTimeout   = 60 * time.Second
	sessionMaxLifetime = 14 * 24 * 60 * 60 // 14d
)

func Start(ctx context.Context, repo *repository.Database, cfg *Configuration) error {
	deviceSrv := service.NewDeviceService(repo)
	subSrv := service.NewSubssService(repo)
	usersSrv := service.NewUsersService(repo)
	episodesSrv := service.NewEpisodesService(repo)
	settingsSrv := service.NewSettingsService(repo)

	// middlewares
	sessionMW, err := newSessionMiddleware(repo)
	if err != nil {
		return err
	}

	authMW := authenticator{usersSrv}

	// routes
	router := chi.NewRouter()
	router.Use(newPromMiddleware("api", nil).Handler)
	router.Use(middleware.RealIP)
	router.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	router.Use(newLogMiddleware(cfg))
	router.Use(newRecoverMiddleware)
	router.Use(middleware.Timeout(connectioTimeout))

	router.Handle("/metrics", newMetricsHandler())

	router.Route("/subscriptions", func(r chi.Router) {
		r.Use(sessionMW)
		r.Use(authMW.handle)
		r.Mount("/", (&simpleResource{cfg, repo, subSrv}).Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Use(sessionMW)
		r.Use(authMW.handle)
		r.Mount("/auth", (&authResource{cfg, usersSrv}).Routes())
		r.Mount("/devices", (&deviceResource{cfg, deviceSrv}).Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{cfg, subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{cfg, episodesSrv}).Routes())
		r.Mount("/updates", (&updatesResource{cfg, subSrv, episodesSrv}).Routes())
		r.Mount("/settings", (&settingsResource{cfg, settingsSrv}).Routes())
	})

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

func writeError(w http.ResponseWriter, r *http.Request, code int, err error) {
	var msg string
	if err == nil {
		msg = http.StatusText(code)
	} else {
		msg = err.Error()
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		res := struct {
			Error string `json:"error"`
		}{msg}

		render.Status(r, code)
		render.JSON(w, r, &res)

		return
	}

	http.Error(w, msg, code)
}
