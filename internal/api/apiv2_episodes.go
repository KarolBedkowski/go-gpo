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

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
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
	ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger,
) {
	var reqData []episode

	if err := render.DecodeJSON(r.Body, &reqData); err != nil {
		logger.Debug().Err(err).Msg("parse json error")
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
			logger.Debug().Interface("req", reqEpisode).Msg("skipped episode")

			continue
		}

		if err := reqEpisode.validate(); err != nil {
			logger.Debug().Err(err).Interface("req", reqEpisode).Msg("validate error")
			http.Error(w, "validate reqData data failed", http.StatusBadRequest)

			return
		}

		actions = append(actions, reqEpisode.toModel())
	}

	logger.Debug().Msgf("uploadEpisodeActions: count=%d, changedurls=%d", len(actions), len(changedurls))

	cmd := command.AddActionCmd{
		UserName: common.ContextUser(ctx),
		Actions:  actions,
	}
	if err := er.episodesSrv.AddAction(ctx, &cmd); err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).
			Err(err).Msg("save episodes error")

		return
	}

	res := struct {
		UpdatedURLs [][]string `json:"update_urls"`
		Timestamp   int64      `json:"timestamp"`
	}{
		Timestamp:   time.Now().UTC().Unix(),
		UpdatedURLs: changedurls,
	}

	render.JSON(w, r, &res)
}

func (er episodesResource) getEpisodeActions(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	podcast := r.URL.Query().Get("podcast")
	device := r.URL.Query().Get("device")
	aggregated := r.URL.Query().Get("aggregated") == "true"

	since, err := getSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msg("parse since parameter to time error")
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

	res, err := er.episodesSrv.GetEpisodes(ctx, &query)
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

	logger.Debug().Msgf("getEpisodeActions: count=%d", len(resp.Actions))

	render.JSON(w, r, &resp)
}

// -----------------------------

type episode struct {
	ts        time.Time `json:"-"`
	Timestamp any       `json:"timestamp"`
	Started   *int32    `json:"started,omitempty"`
	Position  *int32    `json:"position,omitempty"`
	Total     *int32    `json:"total,omitempty"`
	GUID      *string   `json:"guid,omitempty"`
	Podcast   string    `json:"podcast"`
	Episode   string    `json:"episode"`
	Device    string    `json:"device,omitempty"` // Device is optional
	Action    string    `json:"action"`
}

func newEpisodesFromModel(e *model.Episode) episode {
	return episode{
		Podcast:   e.Podcast.URL,
		Episode:   e.URL,
		Device:    e.DeviceName(),
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
		return aerr.New("empty `podcast`").WithTag(aerr.ValidationError)
	}

	if e.Episode == "" {
		return aerr.New("empty `episode`").WithTag(aerr.ValidationError)
	}

	if !validators.IsValidEpisodeAction(e.Action) {
		return aerr.New("invalid `action`").WithTag(aerr.ValidationError)
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
		Podcast:   &model.Podcast{URL: e.Podcast},
		URL:       e.Episode,
		Device:    &model.Device{Name: e.Device},
		Action:    e.Action,
		Timestamp: e.ts,
		Started:   e.Started,
		Position:  e.Position,
		Total:     e.Total,
		GUID:      e.GUID,
	}
}
