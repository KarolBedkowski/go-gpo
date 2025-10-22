//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpodder/internal/repository"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type Configuration struct {
	NoAuth  bool
	Listen  string
	LogBody bool
}

const connectioTimeout = 60 * time.Second

func Start(repo *repository.Repository, cfg *Configuration) error {
	sess, err := session.Sessioner(session.Options{
		Provider:       "file",
		ProviderConfig: "./tmp/",
		CookieName:     "sessionid",
		// Secure:         true,
		// SameSite:       http.SameSiteLaxMode,
		// Maxlifetime: 60 * 60 * 24 * 365,
	})
	if err != nil {
		panic(err.Error())
	}

	deviceSrv := service.NewDeviceService(repo)
	subSrv := service.NewSubssService(repo)
	usersSrv := service.NewUsersService(repo)
	episodesSrv := service.NewEpisodesService(repo)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(hlog.RequestIDHandler("req_id", "Request-Id"))

	if cfg.LogBody {
		router.Use(newLogMiddleware)
	} else {
		router.Use(newSimpleLogMiddleware)
	}

	router.Use(sess)
	router.Use(authenticator{usersSrv}.Authenticate)
	router.Use(newRecoverMiddleware)
	router.Use(middleware.Timeout(connectioTimeout))

	router.Mount("/subscriptions", (&simpleResource{cfg, repo, subSrv}).Routes())

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", (&authResource{cfg, usersSrv}).Routes())
		r.Mount("/devices", (&deviceResource{cfg, deviceSrv}).Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{cfg, subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{cfg, episodesSrv}).Routes())
		r.Mount("/updates", (&updatesResource{cfg, subSrv, episodesSrv}).Routes())
	})

	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("go-gpodder"))
	})

	logRoutes(router)

	if err := http.ListenAndServe(cfg.Listen, router); err != nil {
		return fmt.Errorf("start listen error: %w", err)
	}

	return nil
}

func logRoutes(r chi.Routes) {
	walkFunc := func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		_ = handler
		_ = middlewares
		route = strings.ReplaceAll(route, "/*/", "/")
		log.Debug().Msgf("ROUTE: %s %s", method, route)

		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		log.Error().Err(err).Msg("routers walk error")
	}
}
