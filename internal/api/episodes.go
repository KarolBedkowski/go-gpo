// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type episodesResource struct {
	episodesServ *service.Episodes
}

type episode struct {
	Podcast   string `json:"podcast"`
	Episode   string `json:"episode"`
	Device    string `json:"device"`
	Action    string `json:"action"`
	Timestamp any    `json:"timestamp"`
	Started   *int   `json:"started,omitempty"`
	Position  *int   `json:"position,omitempty"`
	Total     *int   `json:"total,omitempty"`
}

func (e *episode) getTimestamp() (time.Time, error) {
	switch v := e.Timestamp.(type) {
	case int:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case int32:
		return time.Unix(int64(v), 0), nil
	case string:
		if ts, err := parseDate(v); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("cant parse timestamp %v", e.Timestamp)
}

func (er *episodesResource) Routes() chi.Router {
	r := chi.NewRouter()
	// r.Use(AuthenticatedOnly)

	r.Post("/{user:[0-9a-z.-]+}.json", er.uploadEpisodeActions)
	r.Get("/{user:[0-9a-z.-]+}.json", er.getEpisodeActions)
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

	var req []*episode

	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	actions := make([]*model.Episode, 0, len(req))

	for _, r := range req {
		ts, err := r.getTimestamp()
		if err != nil {
			logger.Warn().Err(err).Msgf("parse json error")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		episode := model.Episode{
			Podcast:   r.Podcast,
			Episode:   r.Episode,
			Device:    r.Device,
			Action:    r.Action,
			Timestamp: ts,
			Started:   r.Started,
			Position:  r.Position,
			Total:     r.Total,
		}
		actions = append(actions, &episode)

		logger.Debug().Interface("episode", &episode).Msg("new episode")
	}

	if err = er.episodesServ.SaveEpisodesActions(ctx, user, actions...); err != nil {
		logger.Debug().Interface("req", req).Msg("save episodes error")
		logger.Warn().Err(err).Msg("save episodes error")
		w.WriteHeader(http.StatusInternalServerError)

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

func (er *episodesResource) getEpisodeActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	if suser := userFromContext(ctx); suser != user {
		logger.Warn().Msgf("user %q not match session user: %q", user, suser)
		// w.WriteHeader(http.StatusBadRequest)

		// return
	}

	podcast := r.URL.Query().Get("podcast")
	device := r.URL.Query().Get("device")
	aggregated := r.URL.Query().Get("aggregated") == "true"

	since := time.Time{}
	if s := r.URL.Query().Get("since"); s != "" {
		se, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			logger.Debug().Err(err).Msgf("parse since parameter %q to time error", s)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		since = time.Unix(se, 0)
	}

	res, err := er.episodesServ.GetEpisodesActions(ctx, user, podcast, device, since, aggregated)
	if err != nil {
		logger.Info().Err(err).Msgf("get episodes error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	actions := make([]*episode, 0, len(res))

	for _, r := range res {
		episode := episode{
			Podcast:   r.Podcast,
			Episode:   r.Episode,
			Device:    r.Device,
			Action:    r.Action,
			Timestamp: r.Timestamp.Format("2006-01-02T15:04:05"),
			Started:   r.Started,
			Position:  r.Position,
			Total:     r.Total,
		}
		actions = append(actions, &episode)

		logger.Debug().Interface("episode", &episode).Msg("found episode")
	}

	resp := struct {
		Actions   []*episode `json:"actions"`
		Timestamp int64      `json:"timestamp"`
	}{
		actions, time.Now().Unix(),
	}

	render.JSON(w, r, &resp)
}
