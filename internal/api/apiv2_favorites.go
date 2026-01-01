package api

// apiv2_favorites.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
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
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

// favoritesResource handle request to /api/2/favorites/<user>.json.
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
		Get(`/{user:[\w+.-]+}.json`, srvsupport.WrapNamed(u.getFafovites, "api_favorites"))

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
		logger.WithLevel(aerr.LogLevelForError(err)).Err(err).Msgf("get episodes updates error: %s", err)

		return
	}

	resfavs := common.Map(favorites, newFavoriteFromModel)
	srvsupport.RenderJSON(w, r, resfavs)
}

type favorite struct {
	Released     time.Time `json:"released"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PodcastTitle string    `json:"podcast_title"`
	PodcastURL   string    `json:"podcast_url"`
	Website      string    `json:"website"`
	MygpoLink    string    `json:"mygpo_link"`
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
