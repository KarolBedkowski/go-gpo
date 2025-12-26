package pg

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s Repository) ListSubscribedPodcasts(ctx context.Context, userid int64, since time.Time,
) (model.Podcasts, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("get subscribed podcasts since %s", since)

	res := []PodcastDB{}
	dbctx := db.MustCtx(ctx)

	query := `
		SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at, p.metadata_updated_at,
		coalesce(p.description, '') as description, coalesce(p.website, '') as website
		FROM podcasts p
		WHERE p.user_id = ? AND subscribed `
	args := []any{userid}

	if !since.IsZero() {
		query += " AND p.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	query += " ORDER BY p.title, p.url"

	err := dbctx.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query podcasts failed").WithMeta("user_id", userid, "since", since)
	}

	return podcastsFromDb(res), nil
}

func (s Repository) ListPodcasts(ctx context.Context, userid int64, since time.Time,
) (model.Podcasts, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("get podcasts since %s", since)

	res := []PodcastDB{}
	dbctx := db.MustCtx(ctx)

	query := `
		SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at, p.metadata_updated_at,
		coalesce(p.description, '') as description, coalesce(p.website, '') as website
		FROM podcasts p
		WHERE p.user_id=?`
	args := []any{userid}

	if !since.IsZero() {
		query += " AND p.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	query += " ORDER BY p.title, p.url"

	err := dbctx.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query podcasts failed").WithMeta("user_id", userid, "since", since)
	}

	return podcastsFromDb(res), nil
}

func (s Repository) GetPodcastByID(
	ctx context.Context,
	userid, podcastid int64,
) (*model.Podcast, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).Msg("get podcast")

	dbctx := db.MustCtx(ctx)
	podcast := PodcastDB{}

	err := dbctx.GetContext(ctx, &podcast,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at, p.metadata_updated_at, "+
			"coalesce(p.description, '') as description, coalesce(p.website, '') as website "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.id = ?", userid, podcastid)
	switch {
	case err == nil:
		return podcast.toModel(), nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, common.ErrNoData
	default:
		return nil, aerr.Wrapf(err, "query podcast failed").WithMeta("podcastid", podcastid)
	}
}

func (s Repository) GetPodcast(
	ctx context.Context,
	userid int64,
	podcasturl string,
) (*model.Podcast, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("podcast_url", podcasturl).Msg("get podcast")

	dbctx := db.MustCtx(ctx)
	podcast := PodcastDB{}

	err := dbctx.GetContext(ctx, &podcast,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at, p.metadata_updated_at, "+
			"coalesce(p.description, '') as description, coalesce(p.website, '') as website "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.url = ?", userid, podcasturl)
	switch {
	case err == nil:
		return podcast.toModel(), nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, common.ErrNoData
	default:
		return nil, aerr.Wrapf(err, "query podcast failed").WithMeta("podcasturl", podcasturl)
	}
}

func (s Repository) SavePodcast(ctx context.Context, podcast *model.Podcast) (int64, error) {
	logger := log.Ctx(ctx)
	dbctx := db.MustCtx(ctx)

	metaupdatedat := sql.NullTime{}
	if !podcast.MetaUpdatedAt.IsZero() {
		metaupdatedat = sql.NullTime{Time: podcast.MetaUpdatedAt, Valid: true}
	}

	if podcast.UpdatedAt.IsZero() {
		podcast.UpdatedAt = time.Now().UTC()
	}

	if podcast.ID == 0 {
		logger.Debug().Object("podcast", podcast).Msg("insert podcast")

		res, err := dbctx.ExecContext(
			ctx,
			"INSERT INTO podcasts "+
				"(user_id, title, url, subscribed, created_at, updated_at, metadata_updated_at, website, description) "+
				"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			podcast.User.ID,
			podcast.Title,
			podcast.URL,
			podcast.Subscribed,
			time.Now().UTC(),
			podcast.UpdatedAt,
			metaupdatedat,
			podcast.Website,
			podcast.Description,
		)
		if err != nil {
			return 0, aerr.Wrapf(err, "insert podcast failed").WithMeta("podcast_url", podcast.URL)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, aerr.Wrapf(err, "get last id failed").WithMeta("podcast_url", podcast.URL)
		}

		logger.Debug().Object("podcast", podcast).Msg("podcast created")

		return id, nil
	}

	// update
	logger.Debug().Object("podcast", podcast).Msg("update podcast")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE podcasts SET subscribed=?, title=?, url=?, updated_at=? WHERE id=?",
		podcast.Subscribed, podcast.Title, podcast.URL, podcast.UpdatedAt, podcast.ID)
	if err != nil {
		return 0, aerr.Wrapf(err, "update podcast failed").
			WithMeta("podcast_id", podcast.ID, "podcast_url", podcast.URL)
	}

	return podcast.ID, nil
}

func (s Repository) ListPodcastsToUpdate(ctx context.Context, since time.Time) ([]model.PodcastToUpdate, error) {
	dbctx := db.MustCtx(ctx)

	res := []PodcastToUpdate{}

	// for some reason metadata_updated_at is string, even after datetime function
	err := dbctx.SelectContext(ctx, &res, `
		SELECT p.url as url, min(metadata_updated_at) as metadata_updated_at
		FROM podcasts p
		WHERE p.subscribed AND (metadata_updated_at IS NULL OR metadata_updated_at < ?)
		GROUP by p.url
		`,
		since)
	if err != nil {
		return nil, aerr.Wrapf(err, "get list podcasts to update failed")
	}

	mres := make([]model.PodcastToUpdate, len(res))
	for i, r := range res {
		m, err := r.toModel()
		if err != nil {
			return nil, aerr.Wrapf(err, "convert to model failed").WithMeta("obj", r)
		}

		mres[i] = m
	}

	return mres, nil
}

func (s Repository) UpdatePodcastsInfo(ctx context.Context, update *model.PodcastMetaUpdate) error {
	dbctx := db.MustCtx(ctx)
	logger := log.Ctx(ctx)

	logger.Debug().Object("update", update).Msg("update podcast info")

	var err error

	if update.NotModified {
		_, err = dbctx.ExecContext(ctx,
			"UPDATE podcasts SET metadata_updated_at=? WHERE url=?",
			update.MetaUpdatedAt, update.URL)
	} else {
		_, err = dbctx.ExecContext(ctx,
			`UPDATE podcasts SET title=?, description=?, website=?, metadata_updated_at=? WHERE url=?`,
			update.Title, update.Description, update.Website, update.MetaUpdatedAt, update.URL)
	}

	if err != nil {
		return aerr.Wrapf(err, "update podcasts failed").WithMeta("podcast_update", update)
	}

	return nil
}

func (s Repository) DeletePodcast(ctx context.Context, podcastid int64) error {
	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(ctx, "DELETE  FROM podcasts WHERE id=?", podcastid)
	if err != nil {
		return aerr.Wrapf(err, "delete podcasts failed").WithMeta("podcast_id", podcastid)
	}

	return nil
}
