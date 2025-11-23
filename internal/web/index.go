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
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type indexPage struct {
	episodeSrv *service.EpisodesSrv
	template   templates
}

func newIndexPage(i do.Injector) (indexPage, error) {
	return indexPage{
		episodeSrv: do.MustInvoke[*service.EpisodesSrv](i),
		template:   do.MustInvoke[templates](i),
	}, nil
}

func (i indexPage) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", srvsupport.Wrap(i.indexPage))

	return r
}

const maxLastAction = 25

func (i indexPage) indexPage(ctx context.Context, writer http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	query := query.GetLastEpisodesActionsQuery{
		UserName: user,
		Limit:    maxLastAction,
	}

	lastactions, err := i.episodeSrv.GetLastActions(ctx, &query)
	if err != nil {
		srvsupport.CheckAndWriteError(writer, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get last actions error")

		return
	}

	slices.Reverse(lastactions)

	data := struct {
		LastActions []model.EpisodeLastAction
	}{lastactions}

	if err := i.template.executeTemplate(writer, "index.tmpl", data); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		srvsupport.WriteError(writer, r, http.StatusInternalServerError, "")
	}
}
