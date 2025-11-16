package repository

//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

func (s SqliteRepository) GetUser(ctx context.Context, dbctx DBContext, username string) (UserDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Str("user_name", username).Msg("get user")

	user := UserDB{}

	err := dbctx.GetContext(ctx, &user,
		"SELECT id, username, password, email, name, created_at, updated_at "+
			"FROM users WHERE username=?",
		username)

	switch {
	case err == nil:
		return user, nil
	case errors.Is(err, sql.ErrNoRows):
		return user, ErrNoData
	default:
		return user, aerr.Wrapf(err, "select user failed").WithTag(aerr.InternalError)
	}
}

func (s SqliteRepository) SaveUser(ctx context.Context, dbctx DBContext, user *UserDB) (int64, error) {
	logger := log.Ctx(ctx)

	if user.ID == 0 {
		logger.Debug().Object("user", user).Msg("insert user")

		res, err := dbctx.ExecContext(ctx,
			"INSERT INTO users (username, password, email, name, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			user.UserName, user.Password, user.Email, user.Name, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return 0, aerr.Wrapf(err, "insert user failed").WithTag(aerr.InternalError)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, aerr.Wrapf(err, "get insert id failed").WithTag(aerr.InternalError)
		}

		return id, nil
	}

	// update
	logger.Debug().Object("user", user).Msg("update user")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE users SET password=?, email=?, name=?, updated_at=? WHERE id=?",
		user.Password, user.Email, user.Name, time.Now().UTC(), user.ID)
	if err != nil {
		return 0, aerr.Wrapf(err, "update user failed").WithTag(aerr.InternalError)
	}

	return user.ID, nil
}

// ListUsers get all users from database.
func (s SqliteRepository) ListUsers(ctx context.Context, dbctx DBContext, activeOnly bool) ([]UserDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("list users, active_only=%v", activeOnly)

	var users []UserDB

	sql := "SELECT id, username, password, email, name, created_at, updated_at FROM users"
	if activeOnly {
		sql += " WHERE password != 'LOCKED'"
	}

	sql += " ORDER BY username"

	err := dbctx.SelectContext(ctx, &users, sql)
	if err != nil {
		return nil, aerr.Wrapf(err, "select users failed").WithTag(aerr.InternalError)
	}

	return users, nil
}

// DeleteUser and all related objects.
func (s SqliteRepository) DeleteUser(ctx context.Context, dbctx DBContext, userid int64) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("delete user")

	_, err := dbctx.ExecContext(ctx,
		"DELETE FROM episodes WHERE podcast_id IN (SELECT id FROM podcasts WHERE user_id=?)",
		userid)
	if err != nil {
		return aerr.Wrapf(err, "delete episodes failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM settings WHERE user_id=?", userid); err != nil {
		return aerr.Wrapf(err, "delete settings failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM podcasts WHERE user_id=?", userid); err != nil {
		return aerr.Wrapf(err, "delete podcasts failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM devices WHERE user_id=?", userid); err != nil {
		return aerr.Wrapf(err, "delete devices failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM users WHERE id=?", userid); err != nil {
		return aerr.Wrapf(err, "delete user failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	return nil
}
