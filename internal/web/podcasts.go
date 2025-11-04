package web

//
// podcasts.go
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
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type podcastPages struct {
	podcastsSrv *service.Podcasts
	template    templates
}

func newPodcastPages(i do.Injector) (podcastPages, error) {
	return podcastPages{
		podcastsSrv: do.MustInvoke[*service.Podcasts](i),
		template:    do.MustInvoke[templates](i),
	}, nil
}

func (p podcastPages) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get(`/`, internal.Wrap(p.list))

	return r
}

func (p podcastPages) list(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	podcasts, err := p.podcastsSrv.GetUserPodcasts(ctx, user)
	if err != nil {
		if internal.CheckAndWriteError(w, r, err) {
			logger.Warn().Err(err).Str("mod", "web").Msg("get user podcasts error")
		} else {
			logger.Debug().Err(err).Str("mod", "web").Msg("get user podcasts error")
		}

		return
	}

	data := struct {
		Podcasts []model.Podcast
	}{
		Podcasts: podcasts,
	}

	if err := p.template.executeTemplate(w, "podcasts.tmpl", &data); err != nil {
		logger.Error().Err(err).Str("mod", "web").Msg("execute template error")
		internal.WriteError(w, r, http.StatusInternalServerError, "")
	}
}
