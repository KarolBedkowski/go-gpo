package api

// apiv2_favorites.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
// GET /api/2/favorites/(username).json

import (
	"context"
	"net/http"

	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type favoritesResource struct {
	episodesSrv *service.Episodes
}

func newFavoritesResource(i do.Injector) (favoritesResource, error) {
	return favoritesResource{
		episodesSrv: do.MustInvoke[*service.Episodes](i),
	}, nil
}

func (u favoritesResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.json`, internal.Wrap(u.getFafovites))

	return r
}

func (u favoritesResource) getFafovites(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := internal.ContextUser(ctx)

	favorites, err := u.episodesSrv.GetFavorites(ctx, user)
	if err != nil {
		internal.CheckAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Str("mod", "api").Msg("get episodes updates error")

		return
	}

	render.JSON(w, r, &favorites)
}
