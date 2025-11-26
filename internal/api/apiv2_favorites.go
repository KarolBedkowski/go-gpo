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
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	"gitlab.com/kabes/go-gpo/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type favoritesResource struct {
	episodesSrv *service.EpisodesSrv
}

func newFavoritesResource(i do.Injector) (favoritesResource, error) {
	return favoritesResource{
		episodesSrv: do.MustInvoke[*service.EpisodesSrv](i),
	}, nil
}

func (u favoritesResource) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.With(checkUserMiddleware).
		Get(`/{user:[\w+.-]+}.json`, srvsupport.Wrap(u.getFafovites))

	return r
}

func (u favoritesResource) getFafovites(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	user := common.ContextUser(ctx)

	favorites, err := u.episodesSrv.GetFavorites(ctx, user)
	if err != nil {
		checkAndWriteError(w, r, err)
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msg("get episodes updates error")

		return
	}

	resfavs := common.Map(favorites, newFavoriteFromModel)
	render.JSON(w, r, resfavs)
}

type favorite struct {
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PodcastTitle string    `json:"podcast_title"`
	PodcastURL   string    `json:"podcast_url"`
	Website      string    `json:"website"`
	MygpoLink    string    `json:"mygpo_link"`
	Released     time.Time `json:"released"`
}

func newFavoriteFromModel(f *model.Favorite) favorite {
	return favorite{
		Title:        f.Title,
		URL:          f.URL,
		PodcastTitle: f.PodcastTitle,
		PodcastURL:   f.PodcastURL,
		Website:      f.Website,
		MygpoLink:    f.MygpoLink,
		Released:     f.Released,
	}
}
