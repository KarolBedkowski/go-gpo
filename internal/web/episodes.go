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
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type episodePages struct {
	episodeSrv *service.Episodes
	template   templates
}

func (e episodePages) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get(`/`, internal.Wrap(e.list))

	return r
}

func (e episodePages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	podcast := r.URL.Query().Get("podcast")
	if podcast == "" {
		logger.Debug().Msgf("empty podcast")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	episodes, err := e.episodeSrv.GetPodcastEpisodes(ctx, user, podcast, "")
	if err != nil {
		logger.Error().Err(err).Msg("get list devices error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)

		return
	}

	data := struct {
		Episodes []model.Episode
	}{
		Episodes: episodes,
	}

	if err := e.template.executeTemplate(w, "episodes.tmpl", &data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		internal.WriteError(w, r, http.StatusInternalServerError, nil)
	}
}
