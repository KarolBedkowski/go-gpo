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

	podcastid, err := queryParamAsInt(r, "podcast")
	if err != nil {
		logger.Debug().Err(err).Msgf("web.Episodes: bad request: invalid podcast; error=%q", err)
		e.renderer.WriteBadRequestError(w, r, "Invalid podcast.")

		return
	}

	query := query.GetEpisodesByPodcastQuery{
		UserName:   user,
		PodcastID:  podcastid,
		Aggregated: true,
	}

	episodes, err := e.episodeSrv.GetEpisodesByPodcast(ctx, &query)
	if err != nil {
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Episodes: get podcast episodes user_name=%s error=%q", user, err)
		e.renderer.WriteError(w, r, err)

		return
	}

	e.renderer.WritePage(w, r, &nt.EpisodesPage{Episodes: episodes})
}
