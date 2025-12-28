package pg

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

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s Repository) GetEpisode(
	ctx context.Context,
	userid, podcastid int64,
	episode string,
) (*model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).Str("episode", episode).
		Msgf("get episode")

	query := `
		SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total,
			e.created_at, e.updated_at, e.device_id,
			p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			d.name AS "device.name", d.id AS "device.id"
		FROM episodes e
		JOIN podcasts p on p.id = e.podcast_id
		LEFT JOIN devices d on d.id = e.device_id
		WHERE p.user_id=$1 AND e.podcast_id = $2 and (e.url = $3 or e.guid = $4)`

	res := EpisodeDB{}
	dbctx := db.MustCtx(ctx)

	err := dbctx.GetContext(ctx, &res, query, userid, podcastid, episode, episode)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episode failed").WithTag(aerr.InternalError)
	}

	return res.toModel(), nil
}

// ListEpisodeActions for user, and optionally for device and podcastid.
// If deviceid is given, return actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
// When aggregate get only last action for each episode.
func (s Repository) ListEpisodeActions(
	ctx context.Context,
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	aggregated bool,
	lastelements uint,
) ([]model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("get episodes since=%s aggregated=%v", since, aggregated)

	// ? because of rebind
	query := `
		SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, e.guid,
			e.created_at, e.updated_at, e.device_id,
			p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			d.name AS "device.name", d.id AS "device.id"
		FROM episodes e
		JOIN podcasts p ON p.id = e.podcast_id
		LEFT JOIN devices d ON d.id=e.device_id
		WHERE p.user_id=?`
	args := []any{userid}
	dbctx := db.MustCtx(ctx)

	if !since.IsZero() {
		query += " AND e.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

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

	res := []EpisodeDB{}

	err := dbctx.SelectContext(ctx, &res, sqlx.Rebind(sqlx.DOLLAR, query), args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	logger.Debug().Msgf("get episodes - found %d episodes", len(res))

	if aggregated {
		// aggregation is rarely use so it's ok to get all episodes and aggregate it outside db.
		res = aggregateEpisodes(res)

		logger.Debug().Msgf("get episodes - aggregate %d episodes", len(res))
	}

	// sorting by ts asc
	slices.Reverse(res)

	return episodesFromDb(res), nil
}

func (s Repository) ListFavorites(ctx context.Context, userid int64) ([]model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("get favorites")

	query := `
		SELECT e.id, e.podcast_id, e.url, e.title, e.guid, e.created_at, e.updated_at,
			p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id"
		FROM episodes e
		JOIN podcasts p ON p.id = e.podcast_id
		JOIN settings s ON s.episode_id = e.id
		WHERE p.user_id=$1 AND s.scope = 'episode' and s.key = 'is_favorite' `

	res := []EpisodeDB{}
	dbctx := db.MustCtx(ctx)

	err := dbctx.SelectContext(ctx, &res, query, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError)
	}

	return episodesFromDb(res), nil
}

func (s Repository) GetLastEpisodeAction(ctx context.Context,
	userid, podcastid int64, excludeDelete bool,
) (*model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).
		Msgf("get last episode action excludeDelete=%v", excludeDelete)

	query := `
		SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total,
			e.created_at, e.updated_at, e.device_id,
			p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			d.name AS "device.name", d.id AS "device.id"
		FROM episodes e
		JOIN podcasts p ON p.id = e.podcast_id
		LEFT JOIN devices d ON d.id=e.device_id
		WHERE p.user_id=$1 AND e.podcast_id = $2 `

	if excludeDelete {
		query += " AND e.action != 'delete' "
	}

	query += "ORDER BY e.updated_at DESC LIMIT 1"

	dbctx := db.MustCtx(ctx)
	res := EpisodeDB{}

	err := dbctx.GetContext(ctx, &res, query, userid, podcastid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, common.ErrNoData
	} else if err != nil {
		return nil, aerr.Wrapf(err, "query episode failed").WithTag(aerr.InternalError)
	}

	logger.Debug().Object("episode", &res).Msg("loaded episode")

	return res.toModel(), nil
}

func (s Repository) SaveEpisode(ctx context.Context, userid int64, episodes ...model.Episode) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("save episode")

	dbctx := db.MustCtx(ctx)

	stmt, err := dbctx.PrepareContext(ctx, `
		INSERT INTO episodes (podcast_id, device_id, title, url, action, started, position, total,
			guid, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
	)
	if err != nil {
		return aerr.Wrapf(err, "prepare insert episode stmt failed").WithTag(aerr.InternalError)
	}

	defer stmt.Close()

	for _, episode := range episodes {
		logger.Debug().Object("episode", &episode).Msg("save episode")

		deviceid := sql.NullInt64{}
		if episode.Device != nil {
			deviceid.Valid = true
			deviceid.Int64 = episode.Device.ID
		}

		if episode.Timestamp.IsZero() {
			episode.Timestamp = time.Now().UTC()
		}

		_, err := stmt.ExecContext(
			ctx,
			episode.Podcast.ID,
			deviceid,
			episode.Title,
			episode.URL,
			episode.Action,
			episode.Started,
			episode.Position,
			episode.Total,
			episode.GUID,
			episode.Timestamp,
			episode.Timestamp,
		)
		if err != nil {
			return aerr.Wrapf(err, "insert episode failed").WithTag(aerr.InternalError).
				WithMeta("podcast_id", episode.Podcast.ID, "episode_url", episode.URL)
		}
	}

	return nil
}

func (s Repository) UpdateEpisodeInfo(ctx context.Context, episodes ...model.Episode) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int("num", len(episodes)).Msg("update episode meta")

	dbctx := db.MustCtx(ctx)

	stmt, err := dbctx.PrepareContext(ctx, `
		UPDATE episodes
		SET title=coalesce($1, title), guid=coalesce($2, guid)
		WHERE url=$3`,
	)
	if err != nil {
		return aerr.Wrapf(err, "prepare update episode stmt failed").WithTag(aerr.InternalError)
	}

	defer stmt.Close()

	for _, episode := range episodes {
		logger.Debug().Str("episode_url", episode.URL).
			Str("episode_title", episode.Title).
			Any("episode_guid", episode.GUID).
			Msg("update episode")

		_, err := stmt.ExecContext(ctx, episode.Title, episode.GUID, episode.URL)
		if err != nil {
			return aerr.Wrapf(err, "update episode failed").WithTag(aerr.InternalError).
				WithMeta("episode_url", episode.URL, "episode_title", episode.Title, "episode_guid", episode.GUID)
		}
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
