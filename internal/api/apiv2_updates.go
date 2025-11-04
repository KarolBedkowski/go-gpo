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
	"github.com/samber/do/v2"
)

type updatesResource struct {
	subsSrv     *service.Subs
	episodesSrv *service.Episodes
}

func newUpdatesResource(i do.Injector) (updatesResource, error) {
	return updatesResource{
		subsSrv:     do.MustInvoke[*service.Subs](i),
		episodesSrv: do.MustInvoke[*service.Episodes](i),
	}, nil
}

func (u updatesResource) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(AuthenticatedOnly)

	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(u.getUpdates))

	return r
}

func (u updatesResource) getUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	since, err := internal.GetSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("parse since error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	includeActions := chi.URLParam(r, "include_actions") == "true"

	added, removed, err := u.subsSrv.GetSubscriptionChanges(ctx, user, deviceid, since)
	if err != nil {
		if internal.CheckAndWriteError(w, r, err) {
			logger.Warn().Err(err).Str("mod", "api").Msg("get subscription changes error")
		} else {
			logger.Debug().Err(err).Str("mod", "api").Msg("get subscription changes error")
		}

		return
	}

	updates, err := u.episodesSrv.GetEpisodesUpdates(ctx, user, deviceid, since, includeActions)
	if err != nil {
		if internal.CheckAndWriteError(w, r, err) {
			logger.Warn().Err(err).Str("mod", "api").Msg("get episodes updates error")
		} else {
			logger.Debug().Err(err).Str("mod", "api").Msg("get episodes updates error")
		}

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
