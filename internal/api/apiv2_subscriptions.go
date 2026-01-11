package api

// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/formats"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// subscriptionsResource handle request to /api/2/subscriptions resource.
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
		Get(`/{user:[\w+.-]+}.opml`,
			srvsupport.WrapNamed(sr.userSubscriptions, "api_subs_user"))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`,
			srvsupport.WrapNamed(sr.devSubscriptions, "api_subs_userdev"))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`,
			srvsupport.WrapNamed(sr.uploadSubscriptionChanges, "api_subs_userdev_post"))

	return router
}

func (sr subscriptionsResource) devSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)
	now := time.Now()

	sinceTS, err := getSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("SubscriptionsResource: parse since=%q to time error=%q",
			r.URL.Query().Get("since"), err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	q := query.GetSubscriptionChangesQuery{UserName: user, DeviceName: devicename, Since: sinceTS}

	state, err := sr.subsSrv.GetSubscriptionChanges(ctx, &q)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("SubscriptionsResource: get user_name=%s device_name=%s subscriptions changes error=%q",
				user, devicename, err)

		return
	}

	res := struct {
		Add       []string `json:"add"`
		Remove    []string `json:"remove"`
		Timestamp int64    `json:"timestamp"`
	}{
		Add:       state.AddedURLs(),
		Remove:    state.RemovedURLs(),
		Timestamp: now.UTC().Unix(),
	}

	if e := logger.Debug(); e.Enabled() {
		e.Msgf("SubscriptionsResource: subscriptions result: added=%d removed=%d ts=%d",
			len(res.Add), len(res.Remove), res.Timestamp)
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
	user := common.ContextUser(ctx)

	subs, err := sr.subsSrv.GetUserSubscriptions(ctx, &query.GetUserSubscriptionsQuery{UserName: user})
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("SubscriptionsResource: get user_name=%s subscriptions error=%q", user, err)

		return
	}

	logger.Debug().Msgf("SubscriptionsResource: subscriptions count=%d", len(subs))

	o := formats.NewOPML("go-gpo")
	for _, s := range subs {
		o.AddRSS(s.URL, s.Title, s.Title)
	}

	w.WriteHeader(http.StatusOK)
	render.XML(w, r, &o)
}

func (sr subscriptionsResource) uploadSubscriptionChanges(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)
	now := time.Now()
	changes := struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}{}

	if err := render.DecodeJSON(r.Body, &changes); err != nil {
		logger.Debug().Err(err).Msgf("SubscriptionsResource: parse json error=%q", err)
		writeError(w, r, http.StatusBadRequest)

		return
	}

	cmd := command.ChangeSubscriptionsCmd{
		UserName:   user,
		DeviceName: devicename,
		Add:        changes.Add,
		Remove:     changes.Remove,
		Timestamp:  now.UTC(),
	}

	if e := logger.Debug(); e.Enabled() {
		e.Msgf("SubscriptionsResource: uploadSubscription add=%d, remove=%d", len(cmd.Add), len(cmd.Remove))
	}

	res, err := sr.subsSrv.ChangeSubscriptions(ctx, &cmd)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("SubscriptionsResource: update device subscription changes error=%q", err)

		return
	}

	resp := struct {
		UpdatedURLs [][]string `json:"update_urls"`
		Timestamp   int64      `json:"timestamp"`
	}{
		Timestamp:   now.Unix(),
		UpdatedURLs: res.ChangedURLs,
	}

	render.JSON(w, r, &resp)
}

// -----------------------------
