//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
)

type Configuration struct {
	NoAuth  bool
	Listen  string
	LogBody bool
}

type API struct {
	simpleResource        simpleResource
	authResource          authResource
	deviceResource        deviceResource
	subscriptionsResource subscriptionsResource
	episodesResource      episodesResource
	updatesResource       updatesResource
	settingsResource      settingsResource
}

func New(i do.Injector) (API, error) {
	return API{
		simpleResource:        do.MustInvoke[simpleResource](i),
		authResource:          do.MustInvoke[authResource](i),
		deviceResource:        do.MustInvoke[deviceResource](i),
		subscriptionsResource: do.MustInvoke[subscriptionsResource](i),
		episodesResource:      do.MustInvoke[episodesResource](i),
		updatesResource:       do.MustInvoke[updatesResource](i),
		settingsResource:      do.MustInvoke[settingsResource](i),
	}, nil
}

func (a *API) Routes() *chi.Mux {
	router := chi.NewRouter()
	router.Route("/subscriptions", func(r chi.Router) {
		r.Mount("/", a.simpleResource.Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", a.authResource.Routes())
		r.Mount("/devices", a.deviceResource.Routes())
		r.Mount("/subscriptions", a.subscriptionsResource.Routes())
		r.Mount("/episodes", a.episodesResource.Routes())
		r.Mount("/updates", a.updatesResource.Routes())
		r.Mount("/settings", a.settingsResource.Routes())
	})

	return router
}
