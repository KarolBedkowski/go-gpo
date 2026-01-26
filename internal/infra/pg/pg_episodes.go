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
	"github.com/rs/zerolog"
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
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("pg.Repository: get episodes since=%s", since)

	var (
		query string
		args  []any
	)

	switch {
	case aggregated && deviceid != nil:
		query, args = s.listEpisodeActionsAggregatedDev(userid, *deviceid, podcastid, since, limit, inverse)
	case aggregated:
		query, args = s.listEpisodeActionsAggregated(userid, podcastid, since, limit, inverse)
	default:
		query, args = s.listEpisodeActions(userid, deviceid, podcastid, since, limit, inverse)
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

	logger.Debug().Msgf("pg.Repository: ListEpisodeActions - found=%d", len(res.Episodes))

	return res.Episodes, nil
}

// ListEpisodeActions return query for all actions for podcasts of user, and optionally for device
// and podcastid.
// If deviceid is given, return actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
func (s Repository) listEpisodeActions(
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	limit uint, inverse bool,
) (string, []any) {
	// ? because of rebind
	query := `
		SELECT e.id, e.podcast_id, e.url, e.title, eh.action, eh.started, eh.position, eh.total, e.guid,
			eh.created_at, eh.updated_at, eh.device_id,
			p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			d.name AS "device.name", d.id AS "device.id"
		FROM episodes e
		JOIN podcasts p ON p.id = e.podcast_id
		JOIN episodes_hist eh ON eh.episode_id = e.id
		LEFT JOIN devices d ON d.id=eh.device_id
		WHERE p.user_id=?`
	args := []any{userid}

	if !since.IsZero() {
		query += " AND eh.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	if deviceid != nil {
		query += " AND (eh.device_id != ? OR eh.device_id is NULL) "
		args = append(args, *deviceid) //nolint:wsl_v5
	}

	if podcastid != nil {
		query += " AND e.podcast_id = ?" //nolint:goconst
		args = append(args, *podcastid)  //nolint:wsl_v5
	}

	query += " ORDER BY eh.updated_at"
	if inverse {
		query += " DESC" //nolint:goconst
	}

	if limit > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(limit), 10)
	}

	return query, args
}

// listEpisodeActionsAggregated return query for last action for each podcast for user, and optionally
// for device and podcastid.
// Episodes are sorted by updated_at asc.
func (s Repository) listEpisodeActionsAggregated(
	userid int64, podcastid *int64,
	since time.Time,
	limit uint, inverse bool,
) (string, []any) {
	// ? because of rebind
	epArgs := ""
	args := []any{userid}

	if !since.IsZero() {
		epArgs += " AND e.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	if podcastid != nil {
		epArgs += " AND e.podcast_id = ?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	query := `
		SELECT p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, e.guid,
			e.created_at, e.updated_at, e.device_id,
			d.name AS "device.name", d.id AS "device.id"
		FROM podcasts p
		JOIN episodes e ON e.podcast_id  = p.id
		LEFT JOIN devices d ON d.id=e.device_id
		WHERE p.user_id = ?
		` + epArgs + `
		ORDER BY e.updated_at`

	if inverse {
		query += " DESC"
	}

	if limit > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(limit), 10)
	}

	return query, args
}

// listEpisodeActionsAggregatedDev return sql for last action for each podcast for user and device
// and optionally podcastid. Actions from OTHER than given devices.
// Episodes are sorted by updated_at asc.
func (s Repository) listEpisodeActionsAggregatedDev(
	userid, deviceid int64, podcastid *int64,
	since time.Time,
	limit uint, inverse bool,
) (string, []any) {
	// ? because of rebind
	epArgs := ""
	args := []any{deviceid, userid}

	if !since.IsZero() {
		epArgs += " AND e.updated_at > ? "
		args = append(args, since) //nolint:wsl_v5
	}

	if podcastid != nil {
		epArgs += " AND e.podcast_id = ?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	query := `
		SELECT p.url AS "podcast.url", p.title AS "podcast.title", p.id AS "podcast.id",
			e.id, e.podcast_id, e.url, e.title , eh.action, eh.started, eh.position, eh.total, e.guid,
			eh.created_at, eh.updated_at, eh.device_id,
			d.name AS "device.name", d.id AS "device.id"
		FROM podcasts p
		JOIN episodes e ON e.podcast_id  = p.id
		JOIN LATERAL (
			SELECT eh.action, eh.started, eh.position, eh.total, eh.created_at, eh.updated_at, eh.device_id
			FROM episodes_hist eh
			WHERE eh.episode_id = e.id AND (e.device_id != ? OR e.device_id is NULL)
			ORDER BY eh.updated_at DESC
			LIMIT 1
		) eh ON true
		LEFT JOIN devices d ON d.id=eh.device_id
		WHERE p.user_id = ? ` + epArgs + `
		ORDER BY eh.updated_at`

	if inverse {
		query += " DESC"
	}

	if limit > 0 {
		query += " LIMIT " + strconv.FormatUint(uint64(limit), 10)
	}

	return query, args
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

// GetLastEpisodeAction return last episode with action for given user and podcast.
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
		WHERE p.user_id=$1 AND e.podcast_id = $2
		ORDER by e.updated_at DESC
		LIMIT 1`

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

	stmthist, err := dbctx.PrepareContext(ctx, `
		INSERT INTO episodes_hist
			(episode_id, device_id, "action", started, "position", total, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
	)
	if err != nil {
		return aerr.Wrapf(err, "prepare insert episode stmt failed").WithTag(aerr.InternalError)
	}

	defer stmthist.Close()

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

		eid, err := insertOrUpdateEpisode(ctx, dbctx, logger, &episode, deviceid)
		if err != nil {
			return err
		}

		logger.Debug().Msgf("insert episode history podcast_id=%d episode_id=%d episode_url=%q action=%q",
			episode.Podcast.ID, eid, episode.URL, episode.Action)

		_, err = stmthist.ExecContext(ctx, eid, deviceid, episode.Action,
			episode.Started, episode.Position, episode.Total,
			episode.Timestamp, episode.Timestamp)
		if err != nil {
			return aerr.Wrapf(err, "insert episode history failed").WithTag(aerr.InternalError).
				WithMeta("podcast_id", episode.Podcast.ID, "episode_url", episode.URL)
		}
	}

	return nil
}

// insertOrUpdateEpisode look for episode in database; if not found - create; otherwise update.
// Return episode id.
// Cant use upsert because of we need check modification date before update.
func insertOrUpdateEpisode(
	ctx context.Context,
	dbctx db.Interface,
	logger *zerolog.Logger,
	episode *model.Episode,
	deviceid sql.NullInt64,
) (int64, error) {
	var epinfo struct {
		ID        int64     `db:"id"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	err := dbctx.GetContext(ctx, &epinfo,
		"SELECT id, updated_at FROM episodes WHERE podcast_id=$1 and (url=$2 or guid=$3)",
		episode.Podcast.ID, episode.URL, episode.GUID,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// new episode, insert
		logger.Debug().Msgf("insert new episode podcast_id=%d episode_url=%q", episode.Podcast.ID, episode.URL)

		err := dbctx.GetContext(
			ctx, &epinfo.ID, `
		INSERT INTO episodes (podcast_id, device_id, title, url, action, started, position, total,
			guid, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
			episode.Podcast.ID,
			deviceid,
			episode.Title,
			episode.URL,
			episode.Action, episode.Started, episode.Position, episode.Total,
			episode.GUID,
			episode.Timestamp, episode.Timestamp,
		)
		if err != nil {
			return 0, aerr.Wrapf(err, "insert episode failed").WithTag(aerr.InternalError).
				WithMeta("podcast_id", episode.Podcast.ID, "episode_url", episode.URL)
		}
	case err != nil:
		return 0, aerr.Wrapf(err, "get episode failed").WithTag(aerr.InternalError).
			WithMeta("podcast_id", episode.Podcast.ID, "episode_url", episode.URL)
	case epinfo.UpdatedAt.Before(episode.Timestamp):
		// update episode
		logger.Debug().Msgf("update episode podcast_id=%d episode_url=%q", episode.Podcast.ID, episode.URL)

		_, err = dbctx.ExecContext(ctx, `
				UPDATE episodes
				SET device_id=$2, "action"=$3, started=$4, "position"=$5, total=$6, updated_at=$7
				WHERE id=$1`,
			epinfo.ID,
			deviceid,
			episode.Action, episode.Started, episode.Position, episode.Total,
			episode.Timestamp)
		if err != nil {
			return 0, aerr.Wrapf(err, "insert episode history failed").WithTag(aerr.InternalError).
				WithMeta("podcast_id", episode.Podcast.ID, "episode_url", episode.URL)
		}
	default:
		logger.Debug().Msgf("skip update episode podcast_id=%d episode_url=%q", episode.Podcast.ID, episode.URL)
	}

	return epinfo.ID, nil
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
