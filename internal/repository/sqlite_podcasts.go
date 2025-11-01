package repository

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
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func (s sqliteRepository) ListSubscribedPodcasts(ctx context.Context, dbctx DBContext, userid int64, since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("get subscribed podcasts since %s", since)

	res := []PodcastDB{}

	err := dbctx.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ? and subscribed", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (s sqliteRepository) ListPodcasts(ctx context.Context, dbctx DBContext, userid int64, since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("get podcasts since %s", since)

	res := []PodcastDB{}

	err := dbctx.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ?", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (s sqliteRepository) GetPodcast(
	ctx context.Context,
	dbctx DBContext,
	userid int64,
	podcasturl string,
) (PodcastDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("podcast_url", podcasturl).Msg("get podcast")

	podcast := PodcastDB{}

	err := dbctx.GetContext(ctx, &podcast,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.url = ?", userid, podcasturl)
	switch {
	case err == nil:
		return podcast, nil
	case errors.Is(err, sql.ErrNoRows):
		return podcast, ErrNoData
	default:
		return podcast, fmt.Errorf("query podcast %q error: %w", podcasturl, err)
	}
}

func (s sqliteRepository) SavePodcast(ctx context.Context, dbctx DBContext, podcast *PodcastDB) (int64, error) {
	logger := log.Ctx(ctx)

	if podcast.ID == 0 {
		logger.Debug().Object("podcast", podcast).Msg("insert podcast")

		res, err := dbctx.ExecContext(
			ctx,
			"INSERT INTO podcasts (user_id, title, url, subscribed, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			podcast.UserID,
			podcast.Title,
			podcast.URL,
			podcast.Subscribed,
			time.Now(),
			time.Now(),
		)
		if err != nil {
			return 0, fmt.Errorf("insert new podcast %q error: %w", podcast.URL, err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id for %q error: %w", podcast.URL, err)
		}

		logger.Debug().Object("podcast", podcast).Msg("podcast created")

		return id, nil
	}

	// update
	logger.Debug().Object("podcast", podcast).Msg("update podcast")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE podcasts SET subscribed=?, title=?, url=?, updated_at=? WHERE id=?",
		podcast.Subscribed, podcast.Title, podcast.URL, time.Now(), podcast.ID)
	if err != nil {
		return 0, fmt.Errorf("update subscriptions %d error: %w", podcast.ID, err)
	}

	return podcast.ID, nil
}
