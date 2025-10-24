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

func (t *Transaction) GetSubscribedPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Time("since", since).Msg("get podcasts")

	res := []PodcastDB{}

	err := t.tx.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ? and subscribed", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	logger.Debug().Msgf("get podcasts: %d", len(res))

	return res, nil
}

func (t *Transaction) GetPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Time("since", since).Msg("get podcasts")

	res := []PodcastDB{}

	err := t.tx.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ?", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (t *Transaction) GetPodcast(ctx context.Context, userid int64, podcasturl string) (PodcastDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Str("podcasturl", podcasturl).Msg("get podcast")

	podcast := PodcastDB{}
	err := t.tx.QueryRowxContext(ctx,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.url = ?", userid, podcasturl).
		StructScan(&podcast)

	switch {
	case err == nil:
		return podcast, nil
	case errors.Is(err, sql.ErrNoRows):
		return podcast, ErrNoData
	default:
		return podcast, fmt.Errorf("query podcast %q error: %w", podcasturl, err)
	}
}

func (t *Transaction) SavePodcast(ctx context.Context, user, device string, podcast ...PodcastDB) error {
	_ = user
	_ = device
	logger := log.Ctx(ctx)

	for _, pod := range podcast {
		logger.Debug().Interface("podcast", pod).Msg("save podcast")

		if _, err := t.savePodcast(ctx, pod); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transaction) savePodcast(ctx context.Context, pod PodcastDB) (int64, error) {
	if pod.UpdatedAt.IsZero() {
		pod.UpdatedAt = time.Now()
	}

	if pod.ID == 0 {
		if pod.CreatedAt.IsZero() {
			pod.CreatedAt = time.Now()
		}

		res, err := t.tx.ExecContext(
			ctx,
			"INSERT INTO podcasts (user_id, title, url, subscribed, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			pod.UserID,
			pod.Title,
			pod.URL,
			pod.Subscribed,
			pod.CreatedAt,
			pod.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("insert new podcast %q error: %w", pod.URL, err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id for %q error: %w", pod.URL, err)
		}

		return id, nil
	}

	// update
	_, err := t.tx.ExecContext(ctx,
		"UPDATE podcasts SET subscribed=?, title=?, url=?, updated_at=? WHERE id=?",
		pod.Subscribed, pod.Title, pod.URL, pod.UpdatedAt, pod.ID)
	if err != nil {
		return 0, fmt.Errorf("update subscriptions %d error: %w", pod.ID, err)
	}

	return pod.ID, nil
}

func (t *Transaction) createNewPodcast(ctx context.Context, userid int64, url string) (int64, error) {
	podcast := PodcastDB{UserID: userid, URL: url, Subscribed: true}

	id, err := t.savePodcast(ctx, podcast)
	if err != nil {
		return 0, err
	}

	return id, nil
}
