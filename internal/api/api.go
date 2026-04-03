// Package api handle request do api's endpoints.
package api

//
// api.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
)

type Configuration struct {
	Listen  string
	NoAuth  bool
	LogBody bool
}

// API is handler for all api endpoints.
type API struct {
	router *chi.Mux
}

func New(i do.Injector) (API, error) {
	simpleResource := do.MustInvoke[simpleResource](i)
	authResource := do.MustInvoke[authResource](i)
	deviceResource := do.MustInvoke[deviceResource](i)
	subscriptionsResource := do.MustInvoke[subscriptionsResource](i)
	episodesResource := do.MustInvoke[episodesResource](i)
	updatesResource := do.MustInvoke[updatesResource](i)
	settingsResource := do.MustInvoke[settingsResource](i)
	favoritesResource := do.MustInvoke[favoritesResource](i)

	router := chi.NewRouter()
	api := API{router}

	router.Route("/subscriptions", func(r chi.Router) {
		r.Mount("/", simpleResource.Routes())
	})

	router.Route("/api/2", func(r chi.Router) {
		r.Mount("/auth", authResource.Routes())
		r.Mount("/devices", deviceResource.Routes())
		r.Mount("/subscriptions", subscriptionsResource.Routes())
		r.Mount("/episodes", episodesResource.Routes())
		r.Mount("/updates", updatesResource.Routes())
		r.Mount("/settings", settingsResource.Routes())
		r.Mount("/favorites", favoritesResource.Routes())
	})

	router.NotFound(api.notfoundHandler)
	router.MethodNotAllowed(api.methodNotAllowedHandler)

	return api, nil
}

func (a *API) Routes() *chi.Mux {
	return a.router
}

func (a *API) notfoundHandler(w http.ResponseWriter, r *http.Request) {
	srvsupport.WriteSimpleError(w, r, http.StatusNotFound, "")
}

func (a *API) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	srvsupport.WriteSimpleError(w, r, http.StatusNotFound, "")
}
