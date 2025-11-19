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

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
)

func (s SqliteRepository) ListSettings(ctx context.Context,
	userid int64, podcastid, episodeid, deviceid *int64, scope string,
) ([]SettingsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("settings_scope", scope).
		Any("podcast_id", podcastid).Any("episode_id", episodeid).Any("device_id", deviceid).
		Msg("get settings")

	res := []SettingsDB{}
	dbctx := db.MustCtx(ctx)

	query := "SELECT user_id, podcast_id, episode_id, device_id, scope, key, value " +
		"FROM settings WHERE user_id=? AND scope=? AND podcast_id IS ? AND episode_id IS ? and device_id IS ? "
	args := []any{userid, scope, podcastid, episodeid, deviceid}

	if err := dbctx.SelectContext(ctx, &res, query, args...); err != nil {
		return res, aerr.Wrapf(err, "select settings failed")
	}

	return res, nil
}

// GetSettings return setting for user, scope and key. Create empty SettingsDB object when no data found in db.
func (s SqliteRepository) GetSettings(ctx context.Context,
	userid int64, podcastid, episodeid, deviceid *int64, scope, key string,
) (SettingsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("settings_scope", scope).Str("key", key).
		Any("podcast_id", podcastid).Any("episode_id", episodeid).Any("device_id", deviceid).
		Msg("get settings")

	dbctx := db.MustCtx(ctx)
	res := SettingsDB{}

	query := "SELECT user_id, podcast_id, episode_id, device_id, scope, key, value " +
		"FROM settings WHERE user_id=? AND scope=? and key=?"
	args := []any{userid, scope, key}

	if podcastid != nil {
		query += " AND podcast_id=?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	if episodeid != nil {
		query += " AND episode_id=?"
		args = append(args, *episodeid) //nolint:wsl_v5
	}

	if deviceid != nil {
		query += " AND device_id=?"
		args = append(args, *deviceid) //nolint:wsl_v5
	}

	err := dbctx.GetContext(ctx, &res, query, args...)

	if errors.Is(err, sql.ErrNoRows) {
		res.Scope = scope
		res.Key = key
		res.UserID = userid
		res.EpisodeID = episodeid
		res.PodcastID = podcastid
		res.DeviceID = deviceid
	} else if err != nil {
		return res, aerr.Wrapf(err, "select settings failed")
	}

	return res, nil
}

// SaveSettings insert or update setting.
func (s SqliteRepository) SaveSettings(ctx context.Context, sett *SettingsDB) error {
	logger := log.Ctx(ctx)
	logger.Debug().Object("settings", sett).Msg("save settings")

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(
		ctx,
		"DELETE from settings "+
			"WHERE user_id=? AND podcast_id IS ? AND episode_id IS ? AND device_id IS ? AND scope=? and key=?;",
		sett.UserID, sett.PodcastID, sett.EpisodeID, sett.DeviceID, sett.Scope, sett.Key,
	)
	if err != nil {
		return aerr.Wrapf(err, "delete settings error")
	}

	if sett.Value == "" {
		return nil
	}

	// upsert not work well with null columns
	_, err = dbctx.ExecContext(
		ctx,
		"INSERT INTO settings (user_id, podcast_id, episode_id, device_id, scope, key, value) "+
			"VALUES(?, ?, ?, ?, ?, ?, ?)",
		sett.UserID, sett.PodcastID, sett.EpisodeID, sett.DeviceID, sett.Scope, sett.Key, sett.Value,
	)
	if err != nil {
		return aerr.Wrapf(err, "upsert settings error").WithMeta("args", sett)
	}

	return nil
}
