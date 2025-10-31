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
	"gitlab.com/kabes/go-gpo/internal/service"
)

type Configuration struct {
	NoAuth  bool
	Listen  string
	LogBody bool
}

type API struct{ i do.Injector }

func New(i do.Injector) API {
	return API{i}
}

func (a *API) Routes() chi.Router {
	deviceSrv := do.MustInvoke[*service.Device](a.i)
	subSrv := do.MustInvoke[*service.Subs](a.i)
	episodesSrv := do.MustInvoke[*service.Episodes](a.i)
	settingsSrv := do.MustInvoke[*service.Settings](a.i)

	router := chi.NewRouter()
	router.Route("/subscriptions", func(r chi.Router) {
		r.Mount("/", (&simpleResource{subSrv}).Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", (&authResource{}).Routes())
		r.Mount("/devices", (&deviceResource{deviceSrv}).Routes())
		r.Mount("/subscriptions", (&subscriptionsResource{subSrv}).Routes())
		r.Mount("/episodes", (&episodesResource{episodesSrv}).Routes())
		r.Mount("/updates", (&updatesResource{subSrv, episodesSrv}).Routes())
		r.Mount("/settings", (&settingsResource{settingsSrv}).Routes())
	})

	return router
}
