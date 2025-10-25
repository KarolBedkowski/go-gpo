// updates.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type updatesResource struct {
	cfg          *Configuration
	subSrv       *service.Subs
	episodesServ *service.Episodes
}

func (u updatesResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !u.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
		r.Use(checkUserMiddleware)
	}

	r.Get("/{user:[0-9a-z.-]+}/{deviceid:[0-9a-z.-]+}.json", u.getUpdates)
	return r
}

func (u *updatesResource) getUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	since, err := sinceFromParameter(r)
	if err != nil {
		logger.Info().Err(err).Msgf("parse since error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	includeActions := chi.URLParam(r, "include_actions") == "true"

	added, removed, err := u.subSrv.GetSubsciptionChanges(ctx, user, deviceid, since)
	if err != nil {
		logger.Info().Err(err).Msgf("load subscription changes error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	updates, err := u.episodesServ.GetEpisodesUpdates(ctx, user, deviceid, since, includeActions)
	if err != nil {
		logger.Info().Err(err).Msgf("load episodes updates error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	result := struct {
		Add     []*model.Podcast       `json:"add"`
		Remove  []string               `json:"remove"`
		Updates []*model.EpisodeUpdate `json:"updates"`
	}{
		Add:     added,
		Remove:  removed,
		Updates: updates,
	}

	render.JSON(w, r, &result)
}
