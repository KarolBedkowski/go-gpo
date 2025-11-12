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
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type episodePages struct {
	episodeSrv *service.Episodes
	template   templates
}

func newEpisodePages(i do.Injector) (episodePages, error) {
	return episodePages{
		episodeSrv: do.MustInvoke[*service.Episodes](i),
		template:   do.MustInvoke[templates](i),
	}, nil
}

func (e episodePages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, internal.Wrap(e.list))

	return r
}

func (e episodePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	podcast := r.URL.Query().Get("podcast")
	if podcast == "" {
		logger.Debug().Str("mod", "web").Msgf("empty podcast")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	episodes, err := e.episodeSrv.GetPodcastEpisodes(ctx, user, "", podcast)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "web").Msg("get podcast episodes error")

		return
	}

	data := struct {
		Episodes []model.Episode
	}{
		Episodes: episodes,
	}

	if err := e.template.executeTemplate(w, "episodes.tmpl", &data); err != nil {
		logger.Error().Err(err).Str("mod", "web").Msg("execute template error")
		internal.WriteError(w, r, http.StatusInternalServerError, "")
	}
}
