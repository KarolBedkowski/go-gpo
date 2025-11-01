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

type API struct{}

func New() API {
	return API{}
}

func (a *API) Routes(i do.Injector) chi.Router {
	deviceSrv := do.MustInvoke[*service.Device](i)
	subSrv := do.MustInvoke[*service.Subs](i)
	// usersSrv := do.MustInvoke[*service.Users](a.i)
	episodesSrv := do.MustInvoke[*service.Episodes](i)
	settingsSrv := do.MustInvoke[*service.Settings](i)
	// podcastsSrv := do.MustInvoke[*service.Podcasts](i)

	router := chi.NewRouter()
	router.Route("/subscriptions", func(r chi.Router) {
		r.Mount("/", (&simpleResource{subSrv}).Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", authResource{}.Routes())
		r.Mount("/devices", deviceResource{deviceSrv}.Routes())
		r.Mount("/subscriptions", subscriptionsResource{subSrv}.Routes())
		r.Mount("/episodes", episodesResource{episodesSrv}.Routes())
		r.Mount("/updates", updatesResource{subSrv, episodesSrv}.Routes())
		r.Mount("/settings", settingsResource{settingsSrv}.Routes())
	})

	return router
}
