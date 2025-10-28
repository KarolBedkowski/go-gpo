//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"github.com/go-chi/chi/v5"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type Configuration struct {
	NoAuth  bool
	Listen  string
	LogBody bool
}

type API struct {
	deviceSrv   *service.Device
	subSrv      *service.Subs
	usersSrv    *service.Users
	episodesSrv *service.Episodes
	settingsSrv *service.Settings
}

func New(
	deviceSrv *service.Device,
	subSrv *service.Subs,
	usersSrv *service.Users,
	episodesSrv *service.Episodes,
	settingsSrv *service.Settings,
) API {
	return API{
		deviceSrv:   deviceSrv,
		subSrv:      subSrv,
		usersSrv:    usersSrv,
		episodesSrv: episodesSrv,
		settingsSrv: settingsSrv,
	}
}

func (a *API) Routes() chi.Router {
	router := chi.NewRouter()
	router.Route("/subscriptions", func(r chi.Router) {
		r.Mount("/", (&simpleResource{a.subSrv}).Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", (&authResource{a.usersSrv}).Routes())
		r.Mount("/devices", (&deviceResource{a.deviceSrv}).Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{a.subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{a.episodesSrv}).Routes())
		r.Mount("/updates", (&updatesResource{a.subSrv, a.episodesSrv}).Routes())
		r.Mount("/settings", (&settingsResource{a.settingsSrv}).Routes())
	})

	return router
}
