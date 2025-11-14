// subscriptions.go
// /api/2/subscriptions
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/opml"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type subscriptionsResource struct {
	subsSrv *service.SubscriptionsSrv
}

func newSubscriptionsResource(i do.Injector) (subscriptionsResource, error) {
	return subscriptionsResource{
		subsSrv: do.MustInvoke[*service.SubscriptionsSrv](i),
	}, nil
}

func (sr subscriptionsResource) Routes() *chi.Mux {
	router := chi.NewRouter()

	router.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.opml`, internal.Wrap(sr.userSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(sr.devSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(sr.uploadSubscriptionChanges))

	return router
}

func (sr subscriptionsResource) devSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	var sinceTS time.Time

	if since := r.URL.Query().Get("since"); since != "" {
		ts, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			logger.Debug().Err(err).Msgf("parse since=%q error", since)
			w.WriteHeader(http.StatusBadRequest)
		}

		sinceTS = time.Unix(ts, 0)
	}

	state, err := sr.subsSrv.GetSubscriptionChanges(ctx, user, deviceid, sinceTS)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get device subscriptions changes error")

		return
	}

	res := struct {
		Add       []string `json:"add"`
		Remove    []string `json:"remove"`
		Timestamp int64    `json:"timestamp"`
	}{
		Add:       state.AddedURLs(),
		Remove:    state.RemovedURLs(),
		Timestamp: time.Now().Unix(),
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, &res)
}

func (sr subscriptionsResource) userSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	_ = r
	user := internal.ContextUser(ctx)

	subs, err := sr.subsSrv.GetUserSubscriptions(ctx, user, time.Time{})
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user subscriptions error")

		return
	}

	o := opml.NewOPML("go-gpo")
	o.AddURL(subs...)

	result, err := o.XML()
	if err != nil {
		logger.Warn().Err(err).Msg("get opml xml error")
		internal.WriteError(w, r, http.StatusInternalServerError, "")

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func (sr subscriptionsResource) uploadSubscriptionChanges(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	changes := subscriptionChangesRequest{}
	if err := render.DecodeJSON(r.Body, &changes); err != nil {
		logger.Debug().Err(err).Msgf("parse json error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	subChanges := model.NewSubscriptionChanges(changes.Add, changes.Remove)

	if err := subChanges.Validate(); err != nil {
		logger.Debug().Err(err).Msg("validate request error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	if err := sr.subsSrv.ApplySubscriptionChanges(ctx, user, deviceid, &subChanges, time.Now()); err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("update device subscription changes error")

		return
	}

	resp := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		Timestamp:   time.Now().Unix(),
		UpdatedURLs: subChanges.ChangedURLs,
	}

	render.JSON(w, r, &resp)
}

// -----------------------------

type subscriptionChangesRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}
