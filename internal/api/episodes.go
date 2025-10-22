package api

// episodes.go
// /api/2/episodes/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/kabes/go-gpodder/internal"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

// -----------------------------

func (er *episodesResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !er.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
	}

	r.With(checkUserMiddleware).
		Post("/{user:[0-9a-z_.-]+}.json", wrap(er.uploadEpisodeActions))
	r.With(checkUserMiddleware).
		Get("/{user:[0-9a-z_.-]+}.json", wrap(er.getEpisodeActions))

	return r
}

func (er *episodesResource) uploadEpisodeActions(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	var request []episode

	err := render.DecodeJSON(r.Body, &request)
	if err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	actions := make([]model.Episode, 0, len(request))
	changedurls := make([][]string, 0)

	for _, reqEpisode := range request {
		if curls := reqEpisode.sanitize(); len(curls) > 0 {
			changedurls = append(changedurls, curls...)
		}

		// skip invalid (non http*) podcasts)
		if reqEpisode.Podcast == "" {
			logger.Debug().Interface("req", reqEpisode).Msgf("skipped episode")

			continue
		}

		if err := reqEpisode.validate(); err != nil {
			logger.Warn().Err(err).Interface("req", reqEpisode).Msgf("validate error")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		actions = append(actions, reqEpisode.toModel())
	}

	if err = er.episodesServ.SaveEpisodesActions(ctx, user, actions...); err != nil {
		logger.Debug().Interface("req", request).Msg("save episodes error")
		logger.Warn().Err(err).Msg("save episodes error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	res := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		time.Now().Unix(), changedurls,
	}

	render.JSON(w, r, &res)
}

func (er *episodesResource) getEpisodeActions(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)
	podcast := r.URL.Query().Get("podcast")
	device := r.URL.Query().Get("device")
	aggregated := r.URL.Query().Get("aggregated") == "true"

	since, err := sinceFromParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("parse since parameter to time error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	res, err := er.episodesServ.GetEpisodesActions(ctx, user, podcast, device, since, aggregated)
	if err != nil {
		logger.Info().Err(err).Msgf("get episodes error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	actions := make([]episode, 0, len(res))

	for _, r := range res {
		actions = append(actions, newEpisodesFromModel(r))
	}

	resp := struct {
		Actions   []episode `json:"actions"`
		Timestamp int64     `json:"timestamp"`
	}{
		actions, time.Now().Unix(),
	}

	render.JSON(w, r, &resp)
}

// -----------------------------

type episodesResource struct {
	cfg          *Configuration
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

	ts time.Time `json:"-"`
}

func newEpisodesFromModel(e model.Episode) episode {
	return episode{
		Podcast:   e.Podcast,
		Episode:   e.Episode,
		Device:    e.Device,
		Action:    e.Action,
		Timestamp: e.Timestamp.Format("2006-01-02T15:04:05"),
		Started:   e.Started,
		Position:  e.Position,
		Total:     e.Total,
	}
}

func (e *episode) sanitize() [][]string {
	var changes [][]string

	spodcast := SanitizeURL(e.Podcast)
	if spodcast != e.Podcast {
		e.Podcast = spodcast
		if spodcast != "" {
			changes = append(changes, []string{e.Podcast, spodcast})
		}
	}

	sepisode := SanitizeURL(e.Episode)
	if sepisode != e.Episode {
		e.Episode = sepisode
		if spodcast != "" {
			changes = append(changes, []string{e.Episode, sepisode})
		}
	}

	return changes
}

func (e *episode) validate() error {
	if e.Podcast == "" {
		return NewValidationError("empty `podcast`")
	}

	if e.Episode == "" {
		return NewValidationError("empty `episode`")
	}

	var err error

	e.ts, err = parseTimestamp(e.Timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (e *episode) toModel() model.Episode {
	return model.Episode{
		Podcast:   e.Podcast,
		Episode:   e.Episode,
		Device:    e.Device,
		Action:    e.Action,
		Timestamp: e.ts,
		Started:   e.Started,
		Position:  e.Position,
		Total:     e.Total,
	}
}
