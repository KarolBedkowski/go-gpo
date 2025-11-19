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
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/opml"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
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
		Get(`/{user:[\w+.-]+}.opml`, srvsupport.Wrap(sr.userSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`, srvsupport.Wrap(sr.devSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`, srvsupport.Wrap(sr.uploadSubscriptionChanges))

	return router
}

func (sr subscriptionsResource) devSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	devicename := internal.ContextDevice(ctx)

	var sinceTS time.Time

	if since := r.URL.Query().Get("since"); since != "" {
		ts, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			logger.Debug().Err(err).Msgf("parse since=%q error", since)
			w.WriteHeader(http.StatusBadRequest)
		}

		sinceTS = time.Unix(ts, 0)
	}

	q := query.GetSubscriptionChangesQuery{UserName: user, DeviceName: devicename, Since: sinceTS}

	state, err := sr.subsSrv.GetSubscriptionChanges(ctx, &q)
	if err != nil {
		checkAndWriteError(w, r, err)
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
		Timestamp: time.Now().UTC().Unix(),
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

	subs, err := sr.subsSrv.GetUserSubscriptions(ctx, &query.GetUserSubscriptionsQuery{UserName: user})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get user subscriptions error")

		return
	}

	o := opml.NewOPML("go-gpo")
	o.AddURL(subs...)

	result, err := o.XML()
	if err != nil {
		logger.Warn().Err(err).Msg("get opml xml error")
		writeError(w, r, http.StatusInternalServerError)

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
	devicename := internal.ContextDevice(ctx)

	changes := subscriptionChangesRequest{}
	if err := render.DecodeJSON(r.Body, &changes); err != nil {
		logger.Debug().Err(err).Msgf("parse json error")
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   user,
		DeviceName: devicename,
		Add:        changes.Add,
		Remove:     changes.Remove,
		Timestamp:  time.Now().UTC(),
	}

	res, err := sr.subsSrv.ChangeSubscriptions(ctx, &cmd)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("update device subscription changes error")

		return
	}

	resp := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		Timestamp:   time.Now().UTC().Unix(),
		UpdatedURLs: res.ChangedURLs,
	}

	render.JSON(w, r, &resp)
}

// -----------------------------

type subscriptionChangesRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}
