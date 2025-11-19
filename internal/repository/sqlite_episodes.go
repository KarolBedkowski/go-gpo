package repository

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
)

func (s SqliteRepository) GetEpisode(
	ctx context.Context,
	userid, podcastid int64,
	episode string,
) (EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).Str("episode", episode).
		Msgf("get episode")

	// FIXME: or in where

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title " +
		"FROM episodes e " +
		"JOIN podcasts p on p.id = e.podcast_id " +
		"WHERE p.user_id=? AND e.podcast_id = ? and (e.url = ? or e.guid = ?)"

	res := EpisodeDB{}

	dbctx := db.MustCtx(ctx)

	err := dbctx.GetContext(ctx, &res, query, userid, podcastid, episode, episode)
	if err != nil {
		return res, aerr.Wrapf(err, "query episode failed").WithTag(aerr.InternalError)
	}

	return res, nil
}

// ListEpisodeActions for user, and optionally for device and podcastid.
// If deviceid is given, return actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
// When aggregate get only last action for each episode.
func (s SqliteRepository) ListEpisodeActions(
	ctx context.Context,
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	aggregated bool,
	lastelements uint,
) ([]EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("get episodes since=%s aggregated=%v", since, aggregated)

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, e.guid, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title, d.name as device_name " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id " +
		"LEFT JOIN devices d on d.id=e.device_id " +
		"WHERE p.user_id=? AND e.updated_at > ?"
	args := []any{userid, since}
	dbctx := db.MustCtx(ctx)

	if deviceid != nil {
		query += " AND (e.device_id != ? OR e.device_id is NULL) "
		args = append(args, *deviceid) //nolint:wsl_v5
	}

	if podcastid != nil {
		query += " AND e.podcast_id = ?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	query += " ORDER BY e.updated_at DESC"

	if lastelements > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(lastelements), 10)
	}

	logger.Debug().Msgf("get episodes - query=%q, args=%v", query, args)

	res := []EpisodeDB{}

	err := dbctx.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	logger.Debug().Msgf("get episodes - found %d episodes", len(res))

	if aggregated {
		// TODO: refactor; load only last entries for each podcast from db
		res = aggregateEpisodes(res)

		logger.Debug().Msgf("get episodes - aggregate %d episodes", len(res))
	}

	// sorting by ts asc
	slices.Reverse(res)

	return res, nil
}

func (s SqliteRepository) ListFavorites(ctx context.Context, userid int64) ([]EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("get favorites")

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.guid, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id " +
		"JOIN settings s on s.episode_id = e.id " +
		"WHERE p.user_id=? AND s.scope = 'episode' and s.key = 'is_favorite'"

	res := []EpisodeDB{}
	dbctx := db.MustCtx(ctx)

	err := dbctx.SelectContext(ctx, &res, query, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError)
	}

	return res, nil
}

func (s SqliteRepository) GetLastEpisodeAction(ctx context.Context,
	userid, podcastid int64, excludeDelete bool,
) (EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).
		Msgf("get last episode action excludeDelete=%v", excludeDelete)

		// FIXME: or in where

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id " +
		"WHERE p.user_id=? AND e.podcast_id = ? "

	if excludeDelete {
		query += " AND e.action != 'delete' "
	}

	query += "ORDER BY e.updated_at DESC LIMIT 1"

	dbctx := db.MustCtx(ctx)
	res := EpisodeDB{}

	err := dbctx.GetContext(ctx, &res, query, userid, podcastid)
	if errors.Is(err, sql.ErrNoRows) {
		return res, ErrNoData
	} else if err != nil {
		return res, aerr.Wrapf(err, "query episode failed").WithTag(aerr.InternalError)
	}

	return res, nil
}

func (s SqliteRepository) SaveEpisode(ctx context.Context, userid int64, episode ...EpisodeDB) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("save episode")

	for _, eps := range episode {
		logger.Debug().Object("episode", eps).Msg("update episode")

		if err := s.saveEpisode(ctx, eps); err != nil {
			return err
		}
	}

	return nil
}

func (s SqliteRepository) saveEpisode(ctx context.Context, episode EpisodeDB) error {
	logger := log.Ctx(ctx)
	logger.Debug().Object("episode", episode).Msg("save episode")

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(
		ctx,
		"INSERT INTO episodes (podcast_id, device_id, title, url, action, started, position, total, guid, "+
			"created_at, updated_at) "+
			"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		episode.PodcastID,
		episode.DeviceID,
		episode.Title,
		episode.URL,
		episode.Action,
		episode.Started,
		episode.Position,
		episode.Total,
		episode.GUID,
		episode.CreatedAt,
		episode.UpdatedAt,
	)
	if err != nil {
		return aerr.Wrapf(err, "insert episode failed").WithTag(aerr.InternalError).
			WithMeta("podcast_id", episode.PodcastID, "episode_url", episode.URL)
	}

	return nil
}

func aggregateEpisodes(episodes []EpisodeDB) []EpisodeDB {
	res := make([]EpisodeDB, 0, len(episodes))
	seen := make(map[string]struct{})

	// episodes are sorted by ts desc, so get last first
	for _, e := range episodes {
		if _, ok := seen[e.URL]; !ok {
			res = append(res, e)
			seen[e.URL] = struct{}{}
		}
	}

	return res
}
