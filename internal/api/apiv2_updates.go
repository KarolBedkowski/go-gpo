package api

// updates.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
import (
	"context"
	"net/http"
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

// updatesResource handle request to /api/2/updates resource.
type updatesResource struct {
	subsSrv     *service.SubscriptionsSrv
	episodesSrv *service.EpisodesSrv
}

func newUpdatesResource(i do.Injector) (updatesResource, error) {
	return updatesResource{
		subsSrv:     do.MustInvoke[*service.SubscriptionsSrv](i),
		episodesSrv: do.MustInvoke[*service.EpisodesSrv](i),
	}, nil
}

func (u updatesResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware, checkDeviceMiddleware).
		Get(`/{user:[\w+.-]+}/{devicename:[\w.-]+}.json`, srvsupport.WrapNamed(u.getUpdates, "api_updates"))

	return r
}

func (u updatesResource) getUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)
	devicename := common.ContextDevice(ctx)
	includeActions := r.URL.Query().Get("include_actions") == "true"

	since, err := getSinceParameter(r)
	if err != nil {
		logger.Debug().Err(err).Msgf("UpdatesResource: parse since=%q to time error=%`",
			r.URL.Query().Get("since"), err)
		writeError(w, r, http.StatusBadRequest)

		return
	}

	q := query.GetSubscriptionChangesQuery{UserName: user, DeviceName: devicename, Since: since}

	state, err := u.subsSrv.GetSubscriptionChanges(ctx, &q)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("UpdatesResource: get subscription user_name=%s devicename=%s changes error=%q",
				user, devicename, err)

		return
	}

	query := query.GetEpisodeUpdatesQuery{
		UserName:       user,
		Since:          since,
		IncludeActions: includeActions,
		DeviceName:     "", // device is ignored; all devices have the same subscriptions
	}

	updates, err := u.episodesSrv.GetUpdates(ctx, &query)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("UpdatesResource: get episodes updates user_name=%s devicename=%s error=%`", user, devicename, err)

		return
	}

	result := struct {
		Add        []podcast       `json:"add"`
		Remove     []string        `json:"remove"`
		Updates    []episodeUpdate `json:"updates"`
		Timestamps int64           `json:"timestamp"`
	}{
		Add:        common.Map(state.Added, newPodcastFromModel),
		Remove:     state.RemovedURLs(),
		Updates:    common.Map(updates, newEpisodeUpdateFromModel),
		Timestamps: time.Now().UTC().Unix(),
	}

	srvsupport.RenderJSON(w, r, &result)
}

//------------------------------------------------------------------------------

type episodeUpdate struct {
	Released     time.Time `json:"released"`
	Episode      *episode  `json:"episode,omitempty"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PodcastTitle string    `json:"podcast_title"`
	PodcastURL   string    `json:"podcast_url"`
	Website      string    `json:"website"`
	MygpoLink    string    `json:"mygpo_link"`
	Status       string    `json:"status"`
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

//------------------------------------------------------------------------------

type podcast struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Website     string `json:"website"`
	MygpoLink   string `json:"mygpo_link"`
	Subscribers int    `json:"subscribers"`
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
