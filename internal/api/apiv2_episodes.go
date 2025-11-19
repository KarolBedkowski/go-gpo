package api

// episodes.go
// /api/2/episodes/
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"context"
	"net/http"
	"slices"
	"time"

	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	"gitlab.com/kabes/go-gpo/internal/validators"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

// -----------------------------
type episodesResource struct {
	episodesSrv *service.EpisodesSrv
}

func newEpisodesResource(i do.Injector) (episodesResource, error) {
	return episodesResource{
		episodesSrv: do.MustInvoke[*service.EpisodesSrv](i),
	}, nil
}

func (er episodesResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Post(`/{user:[\w+.-]+}.json`, srvsupport.Wrap(er.uploadEpisodeActions))
	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.json`, srvsupport.Wrap(er.getEpisodeActions))

	return r
}

func (er episodesResource) uploadEpisodeActions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	var reqData []episode

	if err := render.DecodeJSON(r.Body, &reqData); err != nil {
		logger.Debug().Err(err).Msgf("parse json error")
		http.Error(w, "invalid reqData data", http.StatusBadRequest)

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
			logger.Debug().Err(err).Interface("req", reqEpisode).Msgf("validate error")
			http.Error(w, "validate reqData data failed", http.StatusBadRequest)

			return
		}

		actions = append(actions, reqEpisode.toModel())
	}

	user := internal.ContextUser(ctx)

	if err := er.episodesSrv.AddAction(ctx, user, actions...); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).
			Err(err).Msg("save episodes error")

		return
	}

	res := struct {
		Timestamp   int64      `json:"timestamp"`
		UpdatedURLs [][]string `json:"update_urls"`
	}{
		time.Now().UTC().Unix(), changedurls,
	}

	render.JSON(w, r, &res)
}

func (er episodesResource) getEpisodeActions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	podcast := r.URL.Query().Get("podcast")
	device := r.URL.Query().Get("device")
	aggregated := r.URL.Query().Get("aggregated") == "true"

	since, err := getSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("parse since parameter to time error")
		writeError(w, r, http.StatusBadRequest)

		return
	}

	query := query.GetEpisodesQuery{
		UserName:   user,
		Podcast:    podcast,
		DeviceName: device,
		Since:      since,
		Aggregated: aggregated,
	}

	res, err := er.episodesSrv.GetActions(ctx, &query)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get episodes actions error")

		return
	}

	resp := struct {
		Actions   []episode `json:"actions"`
		Timestamp int64     `json:"timestamp"`
	}{
		Actions:   common.Map(res, newEpisodesFromModel),
		Timestamp: time.Now().UTC().Unix(),
	}

	render.JSON(w, r, &resp)
}

// -----------------------------

type episode struct {
	Podcast string `json:"podcast"`
	Episode string `json:"episode"`
	// Device is optional
	Device    string  `json:"device,omitempty"`
	Action    string  `json:"action"`
	Timestamp any     `json:"timestamp"`
	Started   *int    `json:"started,omitempty"`
	Position  *int    `json:"position,omitempty"`
	Total     *int    `json:"total,omitempty"`
	GUID      *string `json:"guid,omitempty"`

	ts time.Time `json:"-"`
}

func newEpisodesFromModel(e *model.Episode) episode {
	return episode{
		Podcast:   e.Podcast,
		Episode:   e.Episode,
		Device:    e.Device,
		Action:    e.Action,
		Timestamp: e.Timestamp.Format("2006-01-02T15:04:05"),
		Started:   e.Started,
		Position:  e.Position,
		Total:     e.Total,
		GUID:      e.GUID,

		ts: time.Time{},
	}
}

func (e *episode) sanitize() [][]string {
	var changes [][]string

	spodcast := validators.SanitizeURL(e.Podcast)
	if spodcast != e.Podcast {
		e.Podcast = spodcast
		if spodcast != "" {
			changes = append(changes, []string{e.Podcast, spodcast})
		}
	}

	sepisode := validators.SanitizeURL(e.Episode)
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
		return aerr.NewSimple("empty `podcast`").WithTag(aerr.DataError)
	}

	if e.Episode == "" {
		return aerr.NewSimple("empty `episode`").WithTag(aerr.DataError)
	}

	if !slices.Contains(model.ValidActions, e.Action) {
		return aerr.NewSimple("invalid `action`").WithTag(aerr.DataError)
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
		GUID:      e.GUID,
	}
}
