// updates.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type updatesResource struct {
	subsSrv     *service.Subs
	episodesSrv *service.Episodes
}

func newUpdatesResource(i do.Injector) (updatesResource, error) {
	return updatesResource{
		subsSrv:     do.MustInvoke[*service.Subs](i),
		episodesSrv: do.MustInvoke[*service.Episodes](i),
	}, nil
}

func (u updatesResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{deviceid:[\w.-]+}.json`, internal.Wrap(u.getUpdates))

	return r
}

func (u updatesResource) getUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)
	deviceid := internal.ContextDevice(ctx)

	since, err := internal.GetSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("parse since error")
		internal.WriteError(w, r, http.StatusBadRequest, "")

		return
	}

	includeActions := r.URL.Query().Get("include_actions") == "true"

	added, removed, err := u.subsSrv.GetSubscriptionChanges(ctx, user, deviceid, since)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get subscription changes error")

		return
	}

	updates, err := u.episodesSrv.GetEpisodesUpdates(ctx, user, "", since, includeActions)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get episodes updates error")

		return
	}

	podcasts := make([]podcast, len(added))
	for i, a := range added {
		podcasts[i] = newPodcastFromModel(&a)
	}

	eupdates := make([]episodeUpdate, len(updates))
	for i, u := range updates {
		eupdates[i] = newEpisodeUpdateFromModel(&u)
	}

	result := struct {
		Add        []podcast       `json:"add"`
		Remove     []string        `json:"remove"`
		Updates    []episodeUpdate `json:"updates"`
		Timestamps int64           `json:"timestamp"`
	}{
		Add:        podcasts,
		Remove:     removed,
		Updates:    eupdates,
		Timestamps: time.Now().Unix(),
	}

	render.JSON(w, r, &result)
}

type episodeUpdate struct {
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PodcastTitle string    `json:"podcast_title"`
	PodcastURL   string    `json:"podcast_url"`
	Website      string    `json:"website"`
	MygpoLink    string    `json:"mygpo_link"`
	Released     time.Time `json:"released"`
	Status       string    `json:"status"`

	Episode *episode `json:"episode,omitempty"`
}

func newEpisodeUpdateFromModel(eup *model.EpisodeUpdate) episodeUpdate {
	episodeupdate := episodeUpdate{
		Title:        eup.Title,
		URL:          eup.URL,
		PodcastTitle: eup.PodcastTitle,
		PodcastURL:   eup.PodcastURL,
		Website:      eup.Website,
		MygpoLink:    eup.MygpoLink,
		Released:     eup.Released,
		Status:       eup.Status,
	}
	if eup.Episode != nil {
		ep := newEpisodesFromModel(eup.Episode)
		episodeupdate.Episode = &ep
	}

	return episodeupdate
}

type podcast struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Subscribers int    `json:"subscribers"`
	LogoURL     string `json:"logo_url"`
	Website     string `json:"website"`
	MygpoLink   string `json:"mygpo_link"`
}

func newPodcastFromModel(p *model.Podcast) podcast {
	return podcast{
		Title:       p.Title,
		URL:         p.URL,
		Description: p.Description,
		Subscribers: p.Subscribers,
		LogoURL:     p.LogoURL,
		Website:     p.Website,
		MygpoLink:   p.MygpoLink,
	}
}
