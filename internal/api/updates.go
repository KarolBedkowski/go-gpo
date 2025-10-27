// updates.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

type updatesResource struct {
	cfg          *Configuration
	subSrv       *service.Subs
	episodesServ *service.Episodes
}

func (u *updatesResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !u.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
	}

	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, wrap(u.getUpdates))

	return r
}

func (u *updatesResource) getUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	since, err := sinceFromParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("parse since error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	includeActions := chi.URLParam(r, "include_actions") == "true"

	added, removed, err := u.subSrv.GetSubscriptionChanges(ctx, user, deviceid, since)
	if err != nil {
		logger.Warn().Err(err).Msgf("load subscription changes error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	updates, err := u.episodesServ.GetEpisodesUpdates(ctx, user, deviceid, since, includeActions)
	if err != nil {
		logger.Warn().Err(err).Msgf("load episodes updates error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	result := struct {
		Add        []model.Podcast       `json:"add"`
		Remove     []string              `json:"remove"`
		Updates    []model.EpisodeUpdate `json:"updates"`
		Timestamps int64                 `json:"timestamp"`
	}{
		Add:        added,
		Remove:     removed,
		Updates:    updates,
		Timestamps: time.Now().Unix(),
	}

	render.JSON(w, r, &result)
}
