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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type indexPage struct {
	episodeSrv *service.EpisodesSrv
	renderer   *nt.Renderer
}

func newIndexPage(i do.Injector) (indexPage, error) {
	return indexPage{
		episodeSrv: do.MustInvoke[*service.EpisodesSrv](i),
		renderer:   do.MustInvoke[*nt.Renderer](i),
	}, nil
}

func (i indexPage) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", srvsupport.WrapNamed(i.indexPage, "web_index"))

	return r
}

const maxLastAction = 25

func (i indexPage) indexPage(ctx context.Context, writer http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	user := common.ContextUser(ctx)

	query := query.GetLastEpisodesActionsQuery{
		UserName: user,
		Limit:    maxLastAction,
	}

	lastactions, err := i.episodeSrv.GetLastActions(ctx, &query)
	if err != nil {
		srvsupport.CheckAndWriteError(writer, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).
			Msgf("web.Index: get last actions for user_name=%s error=%q", user, err)

		return
	}

	slices.Reverse(lastactions)

	i.renderer.WritePage(writer, &nt.IndexPage{LastActions: lastactions})
}
