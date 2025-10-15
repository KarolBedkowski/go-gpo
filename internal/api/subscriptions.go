// auth.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/oxtyped/go-opml/opml"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type subscriptionsResource struct {
	subServ *service.Subs
}

func (sr *subscriptionsResource) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(AuthenticatedOnly)

	r.Post("/{user:[0-9a-z.-]+}.opml", sr.userSubscriptions)
	r.Post("/{user:[0-9a-z.-]+}/{deviceid:[0-9a-z.-]+}.opml", sr.devSubscriptions)
	// TODO: other formats
	r.Put("/{user:[0-9a-z.-]+}/{deviceid:[0-9a-z.-]+}.json", sr.uploadSubscriptionsJSON)
	// TODO: other formats
	r.Post("/{user:[0-9a-z.-]+}/{deviceid:[0-9a-z.-]+}.json", sr.uploadSubscriptionChangesJSON)
	return r
}

func (sr *subscriptionsResource) devSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// var sinceTS  time.Time
	// if since := r.URL.Query().Get("since"); since != "" {
	// 	sinceTS = time.
	// }

	subs, err := sr.subServ.GetDeviceSubscriptions(ctx, user, deviceid, time.Time{})
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)

		return
	case errors.Is(err, service.ErrUnknownDevice):
		logger.Info().Msgf("unknown device: %q", deviceid)
		w.WriteHeader(http.StatusBadRequest)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	o := opml.NewOPMLFromBlank("go-gpodder")
	for _, s := range subs {
		o.AddRSSFromURL(s.Podcast, 2*time.Second)
	}

	result, err := o.XML()
	if err != nil {
		logger.Info().Err(err).Msg("get opml xml error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func (sr *subscriptionsResource) userSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	subs, err := sr.subServ.GetUserSubscriptions(ctx, user, time.Time{})
	switch {
	case err == nil:
	case errors.Is(err, service.ErrUnknownUser):
		logger.Info().Msgf("unknown user: %q", user)
		w.WriteHeader(http.StatusBadRequest)

		return
	default:
		logger.Info().Err(err).Msg("update device error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	o := opml.NewOPMLFromBlank("go-gpodder")
	for _, s := range subs {
		o.AddRSSFromURL(s, 2*time.Second)
	}

	result, err := o.XML()
	if err != nil {
		logger.Info().Err(err).Msg("get opml xml error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func (sr *subscriptionsResource) uploadSubscriptionsJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := chi.URLParam(r, "user")
	logger := hlog.FromRequest(r).With().Str("user", user).Logger()

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger = logger.With().Str("device_id", deviceid).Logger()

	var subs []string

	if err := render.DecodeJSON(r.Body, &subs); err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if err := sr.subServ.UpdateDeviceSubscriptions(ctx, user, deviceid, subs, time.Now()); err != nil {
		logger.Debug().Strs("subs", subs).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")

		w.WriteHeader(http.StatusBadRequest)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (sr *subscriptionsResource) uploadSubscriptionChangesJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := chi.URLParam(r, "user")
	logger := hlog.FromRequest(r).With().Str("user", user).Logger()

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deviceid := chi.URLParam(r, "deviceid")
	if deviceid == "" {
		logger.Info().Msgf("empty deviceId")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger = logger.With().Str("device_id", deviceid).Logger()

	var changes struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}

	if err := render.DecodeJSON(r.Body, &changes); err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}
	// TODO: 400 Bad Request – the same feed has been added and removed in the same request
	// TODO: sanitize

	updatedURLs, err := sr.subServ.UpdateDeviceSubscriptionChanges(ctx, user, deviceid, changes.Add, changes.Remove)
	if err != nil {
		logger.Debug().Interface("changes", changes).Msg("update subscriptions data")
		logger.Warn().Err(err).Msg("update subscriptions error")

		w.WriteHeader(http.StatusBadRequest)

		return
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
