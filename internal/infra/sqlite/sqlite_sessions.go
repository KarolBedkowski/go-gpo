package sqlite

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

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

var ErrDuplicatedSID = errors.New("sid already exists")

func (Repository) DeleteSession(ctx context.Context, sid string) error {
	logger := log.Ctx(ctx)
	logger.Debug().Str("sid", sid).Msg("delete session")

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(ctx, "DELETE FROM sessions WHERE key=?", sid)
	if err != nil {
		return aerr.Wrapf(err, "delete session failed").WithMeta("sid", sid)
	}

	return nil
}

func (Repository) SaveSession(ctx context.Context, sid string, data map[any]any) error {
	logger := log.Ctx(ctx)
	logger.Debug().Str("sid", sid).Msg("save session")

	dbctx := db.MustCtx(ctx)

	encoded, err := encodeSession(data)
	if err != nil {
		return err
	}

	_, err = dbctx.ExecContext(ctx,
		"UPDATE sessions SET data=?, created_at=? WHERE key=?",
		encoded, time.Now().UTC(), sid)
	if err != nil {
		return aerr.Wrapf(err, "update session failed").WithMeta("sid", sid)
	}

	return nil
}

func (Repository) RegenerateSession(ctx context.Context, oldsid, newsid string) error {
	logger := log.Ctx(ctx)
	logger.Debug().Str("sid", newsid).Str("old_sid", oldsid).Msg("regenerate session")

	dbctx := db.MustCtx(ctx)

	res, err := dbctx.ExecContext(ctx, "UPDATE sessions SET key=?, created_at=? WHERE key=?",
		newsid, time.Now().UTC(), oldsid)
	if err != nil {
		return aerr.Wrapf(err, "update session key failed").WithMeta("oldsid", oldsid, "newsid", newsid)
	}

	cnt, err := res.RowsAffected()
	switch {
	case err != nil:
		return aerr.Wrapf(err, "update session failed get affected rows").WithMeta("oldsid", oldsid)
	case cnt == 1:
		return nil
	case cnt > 1:
		return aerr.Wrapf(err, "update session - duplicated sessions").WithMeta("oldsid", oldsid)
	}

	// session not exists - insert
	_, err = dbctx.ExecContext(ctx,
		"INSERT INTO sessions(key, data, created_at) VALUES(?, '', ?)",
		newsid, time.Now().UTC())
	if err != nil {
		return aerr.Wrapf(err, "insert new session failed").WithMeta("sid", newsid)
	}

	return nil
}

func (Repository) CountSessions(ctx context.Context) (int, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("count sessions")

	var total int

	dbctx := db.MustCtx(ctx)

	if err := dbctx.GetContext(ctx, &total, "SELECT COUNT(*) AS num FROM sessions"); err != nil {
		return 0, aerr.Wrapf(err, "count sessions failed")
	}

	return total, nil
}

func (Repository) CleanSessions(
	ctx context.Context,
	maxLifeTime, maxLifeTimeForEmpty time.Duration,
) error {
	oldestUsed := time.Now().UTC().Add(-maxLifeTime)
	oldestEmpty := time.Now().UTC().Add(-maxLifeTimeForEmpty)

	logger := log.Ctx(ctx)
	logger.Debug().Msgf("clean sessions (%s, %s)", oldestUsed, oldestEmpty)

	dbctx := db.MustCtx(ctx)

	res, err := dbctx.ExecContext(ctx, "DELETE FROM sessions WHERE created_at < ?", oldestUsed)
	if err != nil {
		logger.Err(err).Msg("error delete old sessions")
	} else if res != nil {
		affected, err := res.RowsAffected()
		if err != nil {
			log.Logger.Error().Err(err).Msg("error delete old sessions - get affected rows")
		} else {
			logger.Debug().Msgf("session removed: %d", affected)
		}
	}

	// remove empty session older than 2 hour
	res, err = dbctx.ExecContext(ctx, "DELETE FROM sessions WHERE created_at < ? AND data is null", oldestEmpty)
	if err != nil {
		logger.Error().Err(err).Msg("error delete old sessions")
	} else if res != nil {
		affected, err := res.RowsAffected()
		if err != nil {
			log.Logger.Error().Err(err).Msg("error delete old empty sessions - get affected rows")
		} else {
			logger.Debug().Msgf("empty session removed: %d", affected)
		}
	}

	return nil
}

func (Repository) ReadOrCreate(
	ctx context.Context,
	sid string,
	maxLifeTime time.Duration,
) (*model.Session, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Str("sid", sid).Msg("read or create session")

	session := model.Session{
		SID:       sid,
		CreatedAt: time.Now().UTC(),
	}

	var data []byte

	dbctx := db.MustCtx(ctx)
	err := dbctx.QueryRowxContext(ctx, "SELECT key, data, created_at FROM sessions WHERE key=?", sid).
		Scan(&session.SID, &data, &session.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		// create empty session
		_, err := dbctx.ExecContext(ctx, "INSERT INTO sessions(key, created_at) VALUES(?, ?)", sid, session.CreatedAt)
		if err != nil {
			return nil, aerr.Wrapf(err, "insert session failed").WithMeta("sid", sid)
		}
	} else if err != nil {
		return nil, aerr.Wrapf(err, "select session failed").WithMeta("sid", sid)
	}

	if session.IsValid(maxLifeTime) {
		session.Data, err = decodeSession(data)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Debug().Str("sid", sid).Object("session", &session).Msg("session expired")

		session.Data = make(map[any]any)
	}

	return &session, nil
}

func (Repository) SessionExists(ctx context.Context, sid string) (bool, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Str("sid", sid).Msg("check session exists")

	dbctx := db.MustCtx(ctx)

	var count int

	err := dbctx.GetContext(ctx, &count, "SELECT 1 FROM sessions where key=?", sid)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, aerr.Wrapf(err, "check session exists failed").WithMeta("sid", sid)
	}

	return true, nil
}

func decodeSession(data []byte) (map[any]any, error) {
	if len(data) == 0 {
		return make(map[any]any), nil
	}

	sessiondata, err := session.DecodeGob(data)
	if err != nil {
		return nil, fmt.Errorf("decode session error: %w", err)
	}

	return sessiondata, nil
}

func encodeSession(data map[any]any) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	encoded, err := session.EncodeGob(data)
	if err != nil {
		return nil, fmt.Errorf("session encode error: %w", err)
	}

	return encoded, nil
}
