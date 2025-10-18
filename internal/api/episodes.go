// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type episodesResource struct {
	subServ *service.Subs
}

func (er *episodesResource) Routes() chi.Router {
	r := chi.NewRouter()
	// r.Use(AuthenticatedOnly)

	r.Post("/{user:[0-9a-z.-]+}.json", er.uploadEpisodeActions)
	return r
}

func (er *episodesResource) uploadEpisodeActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		// w.WriteHeader(http.StatusBadRequest)

		// return
	}

	var req []struct {
		Podcast   string    `json:"podcast"`
		Episode   string    `json:"episode"`
		Device    string    `json:"device"`
		Action    string    `json:"action"`
		Timestamp time.Time `json:"timestamp"`
		Started   int       `json:"started"`
		Position  int       `json:"position"`
		Total     int       `json:"total"`
	}

	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	res := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		Timestamp: time.Now().Unix(),
	}

	render.JSON(w, r, &res)
}
