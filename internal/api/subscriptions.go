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

	"github.com/oxtyped/go-opml/opml"
	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type subscriptionsResource struct {
	cfg     *Configuration
	subServ *service.Subs
}

func (sr *subscriptionsResource) Routes() chi.Router {
	router := chi.NewRouter()
	if !sr.cfg.NoAuth {
		router.Use(AuthenticatedOnly)
	}

	router.With(checkUserMiddleware).
		Get("/{user:[0-9a-z._-]+}.opml", wrap(sr.userSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Get("/{user:[0-9a-z._-]+}/{deviceid:[0-9a-z._-]+}.json", wrap(sr.devSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Put("/{user:[0-9a-z._-]+}/{deviceid:[0-9a-z._-]+}.json", wrap(sr.uploadSubscriptions))
	router.With(checkUserMiddleware, checkDeviceMiddleware).
		Post("/{user:[0-9a-z._-]+}/{deviceid:[0-9a-z._-]+}.json", wrap(sr.uploadSubscriptionChanges))

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
			logger.Info().Err(err).Msgf("parse since=%q error", since)
			w.WriteHeader(http.StatusBadRequest)
		}

		sinceTS = time.Unix(ts, 0)
	}

	added, removed, err := sr.subServ.GetDeviceSubscriptionChanges(ctx, user, deviceid, sinceTS)
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	res := struct {
		Add       []string `json:"add"`
		Remove    []string `json:"remove"`
		Timestamp int64    `json:"timestamp"`
	}{
		Add:       ensureNotNilList(added),
		Remove:    ensureNotNilList(removed),
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
		logger.Info().Msgf("unknown user: %q", user)
		writeError(w, r, http.StatusBadRequest, nil)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	const deadline = 5 * time.Second

	o := opml.NewOPMLFromBlank("go-gpodder")
	for _, s := range subs {
		o.AddRSSFromURL(s, deadline)
	}

	result, err := o.XML()
	if err != nil {
		logger.Info().Err(err).Msg("get opml xml error")
		writeError(w, r, http.StatusInternalServerError, nil)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
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
		logger.Warn().Err(err).Msgf("parse json error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	if err := sr.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subs, time.Now()); err != nil {
		logger.Debug().Strs("subs", subs).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")
		writeError(w, r, http.StatusBadRequest, nil)

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
		logger.Warn().Err(err).Msgf("parse json error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	updatedURLs := changes.sanitize()

	if err := changes.validate(); err != nil {
		logger.Debug().Err(err).Msg("validate request error")
		writeError(w, r, http.StatusBadRequest, nil)

		return
	}

	err := sr.subServ.UpdateDeviceSubscriptionChanges(ctx, user, deviceid, changes.Add, changes.Remove)
	if err != nil {
		logger.Debug().Interface("changes", changes).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")
		writeError(w, r, http.StatusBadRequest, nil)

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
