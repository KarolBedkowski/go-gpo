// subscriptions.go
// /api/2/subscriptions
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/opml"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type subscriptionsResource struct {
	subServ *service.Subs
}

func (sr *subscriptionsResource) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(AuthenticatedOnly)

	router.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.opml`, internal.Wrap(sr.userSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(sr.devSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Put(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(sr.uploadSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Post(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(sr.uploadSubscriptionChanges))

	return router
}

func (sr *subscriptionsResource) devSubscriptions(
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

	added, removed, err := sr.subServ.GetDeviceSubscriptionChanges(ctx, user, deviceid, sinceTS)
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Warn().Msgf("unknown user: %q", user)
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Debug().Msgf("unknown device: %q", deviceid)
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	default:
		logger.Warn().Err(err).Msg("get subscriptions changes error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)

		return
	}

	res := struct {
		Add       []string `json:"add"`
		Remove    []string `json:"remove"`
		Timestamp int64    `json:"timestamp"`
	}{
		Add:       ensureList(added),
		Remove:    ensureList(removed),
		Timestamp: time.Now().Unix(),
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, &res)
}

func (sr *subscriptionsResource) userSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	_ = r
	user := internal.ContextUser(ctx)

	subs, err := sr.subServ.GetUserSubscriptions(ctx, user, time.Time{})
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Warn().Msgf("unknown user: %q", user)
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	default:
		logger.Warn().Err(err).Msg("get user subscriptions error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)

		return
	}

	o := opml.NewOPML("go-gpo")
	o.AddURL(subs...)

	result, err := o.XML()
	if err != nil {
		logger.Warn().Err(err).Msg("get opml xml error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func (sr *subscriptionsResource) uploadSubscriptions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	var subs []string

	if err := render.DecodeJSON(r.Body, &subs); err != nil {
		logger.Debug().Err(err).Msgf("parse json error")
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := sr.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subs, time.Now()); err != nil {
		logger.Debug().Strs("subs", subs).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (sr *subscriptionsResource) uploadSubscriptionChanges(
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
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	updatedURLs := changes.sanitize()

	if err := changes.validate(); err != nil {
		logger.Debug().Err(err).Msg("validate request error")
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	err := sr.subServ.UpdateDeviceSubscriptionChanges(ctx, user, deviceid, changes.Add, changes.Remove)
	if err != nil {
		logger.Debug().Interface("changes", changes).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")
		internal.WriteError(w, r, http.StatusBadRequest, nil)

		return
	}

	if updatedURLs == nil {
		updatedURLs = make([][]string, 0)
	}

	resp := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		Timestamp:   time.Now().Unix(),
		UpdatedURLs: updatedURLs,
	}

	render.JSON(w, r, &resp)
}

// -----------------------------

type subscriptionChangesRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

func (s *subscriptionChangesRequest) validate() error {
	if len(s.Add) == 0 || len(s.Remove) == 0 {
		return nil
	}

	for _, i := range s.Add {
		if slices.Contains(s.Remove, i) {
			return NewValidationError("duplicated url: %s", i)
		}
	}

	return nil
}

func (s *subscriptionChangesRequest) sanitize() [][]string {
	var chAdd, chRem [][]string

	s.Add, chAdd = SanitizeURLs(s.Add)
	s.Remove, chRem = SanitizeURLs(s.Remove)

	changes := make([][]string, 0)
	changes = append(changes, chAdd...)
	changes = append(changes, chRem...)

	return changes
}
