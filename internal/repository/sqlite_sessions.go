package repository

//
// sessions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License.
//
// Based on gitea.com/go-chi/session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

var ErrDuplicatedSID = errors.New("sid already exists")

func (s sqliteRepository) DeleteSession(ctx context.Context, dbctx DBContext, sid string) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Str("sid", sid).Msg("delete session")

	_, err := dbctx.ExecContext(ctx, "DELETE FROM sessions WHERE key=?", sid)
	if err != nil {
		return fmt.Errorf("delete session %q error: %w", sid, err)
	}

	return nil
}

func (s sqliteRepository) SaveSession(ctx context.Context, dbctx DBContext, sid string, data []byte) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Str("sid", sid).Msg("save session")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE sessions SET data=?, created_at=? WHERE key=?",
		data, time.Now(), sid)
	if err != nil {
		return fmt.Errorf("put session into db error: %w", err)
	}

	return nil
}

func (s sqliteRepository) RegenerateSession(ctx context.Context, dbctx DBContext, oldsid, newsid string) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Str("sid", newsid).Str("old_sid", oldsid).Msg("regenerate session")

	res, err := dbctx.ExecContext(ctx, "UPDATE sessions SET key=? WHERE key=?", newsid, oldsid)
	if err != nil {
		return fmt.Errorf("update key for session %q error: %w", oldsid, err)
	}

	cnt, err := res.RowsAffected()
	switch {
	case err != nil:
		return fmt.Errorf("update session %q get affected rows error: %w", oldsid, err)
	case cnt == 1:
		return nil
	case cnt > 1:
		return fmt.Errorf("update session %q - duplicated sessions", oldsid) //nolint: err113
	}

	// session not exists - insert
	_, err = dbctx.ExecContext(ctx,
		"INSERT INTO sessions(key, data, created_at) VALUES(?, '', ?)",
		newsid, time.Now())
	if err != nil {
		return fmt.Errorf("insert new session %q error: %w", newsid, err)
	}

	return nil
}

func (s sqliteRepository) CountSessions(ctx context.Context, dbctx DBContext) (int, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Msg("count sessions")

	var total int

	if err := dbctx.GetContext(ctx, &total, "SELECT COUNT(*) AS num FROM sessions"); err != nil {
		return 0, fmt.Errorf("error counting records: %w", err)
	}

	return total, nil
}

func (s sqliteRepository) CleanSessions(
	ctx context.Context,
	dbctx DBContext,
	maxLifeTime, maxLifeTimeForEmpty time.Duration,
) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Msg("clean sessions")

	_, err := dbctx.ExecContext(ctx,
		"DELETE FROM sessions WHERE created_at < ?",
		time.Now().Add(-maxLifeTime))
	if err != nil {
		log.Logger.Error().Err(err).Msg("error delete old sessions")
	}

	// remove empty session older than 2 hour
	_, err = dbctx.ExecContext(ctx,
		"DELETE FROM sessions WHERE created_at < ? AND data is null",
		time.Now().Add(maxLifeTimeForEmpty))
	if err != nil {
		log.Logger.Error().Err(err).Msg("error delete old sessions")
	}

	return nil
}

func (s sqliteRepository) ReadOrCreate(ctx context.Context, dbctx DBContext, sid string) ([]byte, time.Time, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Str("sid", sid).Msg("read or create session")

	var (
		data      []byte
		createdat = time.Now()
	)

	err := dbctx.QueryRowxContext(ctx, "SELECT data, created_at FROM sessions WHERE key=?", sid).
		Scan(&data, &createdat)
	if errors.Is(err, sql.ErrNoRows) {
		// create empty session
		_, err := dbctx.ExecContext(ctx, "INSERT INTO sessions(key, created_at) VALUES(?, ?)", sid, createdat)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("insert session into db error: %w", err)
		}
	} else if err != nil {
		return nil, time.Time{}, fmt.Errorf("get session data from db error: %w", err)
	}

	return data, createdat, nil
}

func (s sqliteRepository) SessionExists(ctx context.Context, dbctx DBContext, sid string) (bool, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_session").Logger()
	logger.Debug().Msg("count sessions")

	var count int

	err := dbctx.GetContext(ctx, &count, "SELECT 1 FROM sessions where key=?", sid)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("check session %q exists error: %w", sid, err)
	}

	return true, nil
}
