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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type indexPage struct {
	episodeSrv *service.Episodes
	template   templates
}

func newIndexPage(i do.Injector) (indexPage, error) {
	return indexPage{
		episodeSrv: do.MustInvoke[*service.Episodes](i),
		template:   do.MustInvoke[templates](i),
	}, nil
}

func (i indexPage) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", internal.Wrap(i.indexPage))

	return r
}

const maxLastAction = 25

func (i indexPage) indexPage(ctx context.Context, writer http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := internal.ContextUser(ctx)

	lastactions, err := i.episodeSrv.GetLastActions(ctx, user, time.Time{}, maxLastAction)
	if err != nil {
		internal.CheckAndWriteError(writer, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "web").Msg("get last actions error")

		return
	}

	slices.Reverse(lastactions)

	data := struct {
		LastActions []model.Episode
	}{lastactions}

	if err := i.template.executeTemplate(writer, "index.tmpl", data); err != nil {
		logger.Error().Err(err).Str("mod", "web").Msg("execute template error")
		internal.WriteError(writer, r, http.StatusInternalServerError, "")
	}
}
