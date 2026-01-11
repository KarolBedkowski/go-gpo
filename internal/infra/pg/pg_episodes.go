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
		Msgf("pg.Repository: get episode user_id=%d podcast_id=%d episode=%q", userid, podcastid, episode)

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
	aggregated, inverse bool,
	limit uint,
) ([]model.Episode, error) {
	if aggregated {
		return s.listEpisodeActionsAggregated(ctx, userid, deviceid, podcastid, since, limit, inverse)
	}

	return s.listEpisodeActions(ctx, userid, deviceid, podcastid, since, limit, inverse)
}

// ListEpisodeActions return list of all actions for podcasts of user, and optionally for device
// and podcastid.
// If deviceid is given, return actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
func (s Repository) listEpisodeActions(
	ctx context.Context,
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	limit uint, inverse bool,
) ([]model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("pg.Repository: get episodes since=%s", since)

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

	query += " ORDER BY e.updated_at"
	if inverse {
		query += " DESC"
	}

	if limit > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(limit), 10)
	}

	rows, err := dbctx.QueryxContext(ctx, sqlx.Rebind(sqlx.DOLLAR, query), args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	defer rows.Close()

	res := newEpisodeCollector()
	if err := res.loadRows(rows); err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed, load rows error").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	logger.Debug().Msgf("pg.Repository: get episodes - found=%d", len(res.Episodes))

	return res.Episodes, nil
}

// listEpisodeActionsAggregated return list of last action for each podcast for user, and optionally
// for device and podcastid. If deviceid is given, return actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
func (s Repository) listEpisodeActionsAggregated( //nolint:funlen
	ctx context.Context,
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	limit uint, inverse bool,
) ([]model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("pg.Repository: get episodes since=%s aggregated=true", since)

	// ? because of rebind
	epArgs := ""
	args := []any{userid}

	if !since.IsZero() {
		epArgs += " AND e.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	if deviceid != nil {
		epArgs += " AND (e.device_id != ? OR e.device_id is NULL) "
		args = append(args, *deviceid) //nolint:wsl_v5
	}

	if podcastid != nil {
		epArgs += " AND e.podcast_id = ?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	query := `
		WITH pe AS (
			SELECT e.podcast_id, (
				SELECT e2.id
				FROM episodes e2
				WHERE e2.podcast_id = e.podcast_id AND e2.url = e.url
				ORDER BY e2.updated_at
				DESC LIMIT 1
			) AS episode_id
			FROM podcasts p
			JOIN episodes e ON p.id = e.podcast_id
			WHERE p.user_id=? ` + epArgs + `
			GROUP BY e.podcast_id, e.url
		)
		SELECT p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, e.guid,
			e.created_at, e.updated_at, e.device_id,
			d.name AS "device.name", d.id AS "device.id"
		FROM pe
		JOIN podcasts p ON p.id = pe.podcast_id
		JOIN episodes e ON e.id = pe.episode_id
		LEFT JOIN devices d ON d.id=e.device_id
		ORDER BY e.updated_at`

	if inverse {
		query += " DESC"
	}

	if limit > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(limit), 10)
	}

	logger.Debug().Msgf("pg.Repository: get episodes - sql=%s args=%v", query, args)

	dbctx := db.MustCtx(ctx)

	rows, err := dbctx.QueryxContext(ctx, sqlx.Rebind(sqlx.DOLLAR, query), args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	defer rows.Close()

	res := newEpisodeCollector()
	if err := res.loadRows(rows); err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed, load rows error").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	logger.Debug().Msgf("pg.Repository: get episodes - found=%d", len(res.Episodes))

	return res.Episodes, nil
}

func (s Repository) ListFavorites(ctx context.Context, userid int64) ([]model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("pg.Repository: get favorites user_id=%d", userid)

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

	return episodesFromDB(res), nil
}

func (s Repository) GetLastEpisodeAction(ctx context.Context,
	userid, podcastid int64, excludeDelete bool,
) (*model.Episode, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).
		Msgf("pg.Repository: get last episode action userid=%d podcastid=%d excludeDelete=%v",
			userid, podcastid, excludeDelete)

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

	logger.Debug().Object("episode", &res).Msgf("pg.Repository: loaded episode episodeid=%d", res.ID)

	return res.toModel(), nil
}

func (s Repository) SaveEpisode(ctx context.Context, userid int64, episodes ...model.Episode) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).
		Msgf("pg.Repository: save episodes user_id=%d count=%d", userid, len(episodes))

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
		logger.Debug().Object("episode", &episode).
			Msgf("pg.Repository: save episode podcast_id=%d episode_url=%q", episode.Podcast.ID, episode.URL)

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
	logger.Debug().Msgf("pg.Repository: update episode meta count=%d", len(episodes))

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
			Msgf("pg.Repository: update episode episode_url=%q episode_title=%q", episode.URL, episode.Title)

		_, err := stmt.ExecContext(ctx, episode.Title, episode.GUID, episode.URL)
		if err != nil {
			return aerr.Wrapf(err, "update episode failed").WithTag(aerr.InternalError).
				WithMeta("episode_url", episode.URL, "episode_title", episode.Title, "episode_guid", episode.GUID)
		}
	}

	return nil
}
