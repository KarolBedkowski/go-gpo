package web

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type episodePages struct {
	episodeSrv *service.EpisodesSrv
	renderer   *nt.Renderer
}

func newEpisodePages(i do.Injector) (episodePages, error) {
	return episodePages{
		episodeSrv: do.MustInvoke[*service.EpisodesSrv](i),
		renderer:   do.MustInvoke[*nt.Renderer](i),
	}, nil
}

func (e episodePages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.WrapNamed(e.list, "web_episoeds_list"))

	return r
}

func (e episodePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	podcast := r.URL.Query().Get("podcast")
	if podcast == "" {
		logger.Debug().Msgf("web.Episodes: bad request empty podcast user_name=%s", user)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	podcastid, err := strconv.ParseInt(podcast, 10, 32)
	if err != nil {
		logger.Debug().Err(err).Msgf("web.Episodes: bad request: invalid_podcast_id=%q parse error=%q", podcast, err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	query := query.GetEpisodesByPodcastQuery{
		UserName:   user,
		PodcastID:  podcastid,
		Aggregated: true,
	}

	episodes, err := e.episodeSrv.GetEpisodesByPodcast(ctx, &query)
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Episodes: get podcast episodes user_name=%s error=%q", user, err)

		return
	}

	e.renderer.WritePage(w, &nt.EpisodesPage{Episodes: episodes})
}
