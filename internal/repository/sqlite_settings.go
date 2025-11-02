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

	"github.com/rs/zerolog/log"
)

func (s sqliteRepository) GetSettings(ctx context.Context, db DBContext, userid int64, scope, key string,
) (SettingsDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_settings").Logger()
	logger.Debug().Int64("user_id", userid).Str("settings_scope", scope).Str("settings_key", key).Msg("get settings")

	res := SettingsDB{}

	err := db.GetContext(ctx, &res,
		"SELECT user_id, scope, key, value FROM settings WHERE user_id=? AND scope=? and key=?",
		userid, scope, key)

	if errors.Is(err, sql.ErrNoRows) {
		return SettingsDB{
			UserID: userid,
			Scope:  scope,
			Key:    key,
			Value:  "",
		}, nil
	} else if err != nil {
		return res, fmt.Errorf("query settings for user %d scope %q key %q error: %w", userid, scope, key, err)
	}

	return res, nil
}

func (s sqliteRepository) SaveSettings(ctx context.Context, db DBContext, sett *SettingsDB) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_settings").Logger()
	logger.Debug().Object("settings", sett).Msg("save settings")

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO settings (user_id, scope, key, value) VALUES(?, ?, ?, ?) "+
			"ON CONFLICT(user_id, scope, key) DO UPDATE SET value=excluded.value",
		sett.UserID,
		sett.Scope,
		sett.Key,
		sett.Value,
	)
	if err != nil {
		return fmt.Errorf("save settings error: %w", err)
	}

	return nil
}
