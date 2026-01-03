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

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s Repository) GetSettings(ctx context.Context, key *model.SettingsKey) (model.Settings, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Object("key", key).Msgf("pg.Repository: get settings user_id=%d", key.UserID)

	res := []SettingsDB{}
	dbctx := db.MustCtx(ctx)

	query := `
		SELECT user_id, podcast_id, episode_id, device_id, scope, key, value
		FROM settings
		WHERE user_id=$1 AND scope=$2
			AND NOT (podcast_id IS DISTINCT FROM $3)
			AND NOT (episode_id IS DISTINCT FROM $4)
			AND NOT (device_id IS DISTINCT FROM $5)`

	err := dbctx.SelectContext(ctx, &res, query,
		key.UserID, key.Scope, key.PodcastID, key.EpisodeID, key.DeviceID)
	if err != nil {
		return nil, aerr.Wrapf(err, "select settings failed")
	}

	settings := make(map[string]string)
	for _, r := range res {
		settings[r.Key] = r.Value
	}

	return settings, nil
}

// SaveSettings insert or update setting.
func (s Repository) SaveSettings(ctx context.Context, key *model.SettingsKey, value string,
) error {
	logger := log.Ctx(ctx)
	logger.Debug().Object("key", key).Str("value", value).Msgf("pg.Repository: save settings user_id=%d", key.UserID)

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(
		ctx, `
		DELETE FROM settings
		WHERE user_id=$1 AND scope=$2
			AND NOT (podcast_id IS DISTINCT FROM $3)
			AND NOT (episode_id IS DISTINCT FROM $4)
			AND NOT (device_id IS DISTINCT FROM $5)
			AND key=$6`,
		key.UserID, key.Scope, key.PodcastID, key.EpisodeID, key.DeviceID, key.Key,
	)
	if err != nil {
		return aerr.Wrapf(err, "delete settings error")
	}

	if value == "" {
		return nil
	}

	// upsert not work well with null columns
	_, err = dbctx.ExecContext(
		ctx, `
		INSERT INTO settings (user_id, scope, podcast_id, episode_id, device_id, key, value)
		VALUES($1, $2, $3, $4, $5, $6, $7)`,
		key.UserID, key.Scope, key.PodcastID, key.EpisodeID, key.DeviceID, key.Key, value,
	)
	if err != nil {
		return aerr.Wrapf(err, "insert settings error")
	}

	return nil
}

func (Repository) GetAllSettings(ctx context.Context, userid int64) ([]model.UserSettings, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Msgf("pg.Repository: get all user settings user_id=%d", userid)

	dbctx := db.MustCtx(ctx)
	dbsettings := make([]SettingsDB, 0)

	query := `
		SELECT user_id, podcast_id, episode_id, device_id, scope, key, value
		FROM settings WHERE user_id=$1`

	err := dbctx.SelectContext(ctx, &dbsettings, query, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "select settings failed")
	}

	res := make([]model.UserSettings, len(dbsettings))
	for i, s := range dbsettings {
		res[i] = model.UserSettings{
			UserID:    s.UserID,
			PodcastID: sqlNullInt64ToPtr(s.PodcastID),
			EpisodeID: sqlNullInt64ToPtr(s.EpisodeID),
			DeviceID:  sqlNullInt64ToPtr(s.DeviceID),
			Scope:     s.Scope,
			Key:       s.Key,
			Value:     s.Value,
		}
	}

	return res, nil
}

func sqlNullInt64ToPtr(v sql.NullInt64) *int64 {
	if v.Valid {
		return &v.Int64
	}

	return nil
}
