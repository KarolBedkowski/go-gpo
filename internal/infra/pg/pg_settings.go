package pg

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s Repository) GetSettings(ctx context.Context, key *model.SettingsKey) (model.Settings, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Object("key", key).Msg("get settings")

	res := []SettingsDB{}
	dbctx := db.MustCtx(ctx)

	query := `
		SELECT user_id, podcast_id, episode_id, device_id, scope, key, value
		FROM settings WHERE user_id=? AND scope=? AND podcast_id IS ? AND episode_id IS ? and device_id IS ?`

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
	logger.Debug().Object("key", key).Str("value", value).Msg("save settings")

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(
		ctx, `
		DELETE from settings
		WHERE user_id=? AND scope=? AND podcast_id IS ? AND episode_id IS ? AND device_id IS ? AND key=?`,
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
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		key.UserID, key.Scope, key.PodcastID, key.EpisodeID, key.DeviceID, key.Key, value,
	)
	if err != nil {
		return aerr.Wrapf(err, "insert settings error")
	}

	return nil
}
