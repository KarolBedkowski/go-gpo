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
	webroot    string
}

func newEpisodePages(i do.Injector) (episodePages, error) {
	return episodePages{
		episodeSrv: do.MustInvoke[*service.EpisodesSrv](i),
		webroot:    do.MustInvokeNamed[string](i, "server.webroot"),
	}, nil
}

func (e episodePages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, srvsupport.Wrap(e.list))

	return r
}

func (e episodePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	podcast := r.URL.Query().Get("podcast")
	if podcast == "" {
		logger.Debug().Msg("empty podcast")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	podcastid, err := strconv.ParseInt(podcast, 10, 32)
	if err != nil {
		logger.Debug().Err(err).Msg("invalid podcast id")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	query := query.GetEpisodesByPodcastQuery{
		UserName:   user,
		PodcastID:  int32(podcastid),
		Aggregated: true,
	}

	episodes, err := e.episodeSrv.GetEpisodesByPodcast(ctx, &query)
	if err != nil {
		srvsupport.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get podcast episodes error")

		return
	}

	nt.WritePageTemplate(w, &nt.EpisodesPage{Episodes: episodes}, e.webroot)
}
