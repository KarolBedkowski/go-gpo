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

func (t *Transaction) GetSettings(ctx context.Context, userid int64, scope, key string,
) (SettingsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Str("scope", scope).Str("key", key).Msg("get settings")

	res := SettingsDB{}

	err := t.tx.GetContext(ctx, &res,
		"SELECT user_id, scope, key, value "+
			"FROM settings "+
			"WHERE user_id=? AND scope=? and key=?",
		userid, scope, key)

	if errors.Is(err, sql.ErrNoRows) {
		return SettingsDB{
			UserID: userid,
			Scope:  scope,
			Key:    key,
			Value:  "",
		}, nil
	} else if err != nil {
		return res, fmt.Errorf("query settings error: %w", err)
	}

	return res, nil
}

func (t *Transaction) SaveSettings(ctx context.Context, s *SettingsDB) error {
	logger := log.Ctx(ctx)

	logger.Debug().Interface("settings", s).Msg("save settings")
	_, err := t.tx.ExecContext(
		ctx,
		"INSERT INTO settings (user_id, scope, key, value) VALUES(?, ?, ?, ?) "+
			"ON CONFLICT(user_id, scope, key) DO UPDATE SET value=excluded.value",
		s.UserID,
		s.Scope,
		s.Key,
		s.Value,
	)
	if err != nil {
		return fmt.Errorf("save settings error: %w", err)
	}

	return nil
}
