// episodes.go
// /api/2/episodes/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

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

func (e *episode) sanitize() [][]string {
	var changes [][]string

	spodcast := service.SanitizeURL(e.Podcast)
	if spodcast != e.Podcast {
		changes = append(changes, []string{e.Podcast, spodcast})
		e.Podcast = spodcast
	}

	sepisode := service.SanitizeURL(e.Episode)
	if sepisode != e.Episode {
		changes = append(changes, []string{e.Episode, sepisode})
		e.Episode = sepisode
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

	e.ts, err = e.getTimestamp()
	if err != nil {
		return err
	}

	return nil
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

	return time.Time{}, NewParseError("cant parse timestamp %v", e.Timestamp)
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

// -----------------------------

func (er *episodesResource) Routes() chi.Router {
	r := chi.NewRouter()
	if !er.cfg.NoAuth {
		r.Use(AuthenticatedOnly)
		r.Use(checkUserMiddleware)
	}

	r.Post("/{user:[0-9a-z_.-]+}.json", er.uploadEpisodeActions)
	r.Get("/{user:[0-9a-z_.-]+}.json", er.getEpisodeActions)

	return r
}

func (er *episodesResource) uploadEpisodeActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")

	var reqData []episode

	err := render.DecodeJSON(r.Body, &reqData)
	if err != nil {
		logger.Warn().Err(err).Msgf("parse json error")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	actions := make([]model.Episode, 0, len(reqData))
	changedurls := make([][]string, 0)

	for _, reqEpisode := range reqData {
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

		episode := reqEpisode.toModel()

		logger.Debug().Interface("episode", &episode).Msg("new episode")

		actions = append(actions, episode)
	}

	if err = er.episodesServ.SaveEpisodesActions(ctx, user, actions...); err != nil {
		logger.Debug().Interface("req", reqData).Msg("save episodes error")
		logger.Warn().Err(err).Msg("save episodes error")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	res := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		Timestamp:   time.Now().Unix(),
		UpdatedURLs: changedurls,
	}

	render.JSON(w, r, &res)
}

func (er *episodesResource) getEpisodeActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := hlog.FromRequest(r)
	user := chi.URLParam(r, "user")
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
