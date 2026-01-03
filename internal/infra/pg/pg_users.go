package pg

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
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s Repository) GetUser(ctx context.Context, username string) (*model.User, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Str("user_name", username).Msgf("pg.Repository: get user user_name=%s", username)

	dbctx := db.MustCtx(ctx)
	user := UserDB{}

	err := dbctx.GetContext(ctx, &user, `
		SELECT id, username, password, email, name, created_at, updated_at
		FROM users
		WHERE username=$1`,
		username)

	switch {
	case err == nil:
		return user.toModel(), nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, common.ErrNoData
	default:
		return nil, aerr.Wrapf(err, "select user failed").WithTag(aerr.InternalError)
	}
}

func (s Repository) SaveUser(ctx context.Context, user *model.User) (int64, error) {
	logger := log.Ctx(ctx)
	dbctx := db.MustCtx(ctx)

	if user.ID == 0 {
		logger.Debug().Object("user", user).Msgf("pg.Repository: insert user user_name=%s", user.UserName)

		var id int64

		err := dbctx.GetContext(ctx, &id, `
			INSERT INTO users (username, password, email, name, created_at, updated_at)
				VALUES($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			user.UserName, user.Password, user.Email, user.Name, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return 0, aerr.Wrapf(err, "insert user failed").WithTag(aerr.InternalError)
		}

		return id, nil
	}

	// update
	logger.Debug().Object("user", user).Msgf("pg.Repository: update user user_name=%s", user.UserName)

	_, err := dbctx.ExecContext(ctx,
		"UPDATE users SET password=$1, email=$2, name=$3, updated_at=$4 WHERE id=$5",
		user.Password, user.Email, user.Name, time.Now().UTC(), user.ID)
	if err != nil {
		return 0, aerr.Wrapf(err, "update user failed").WithTag(aerr.InternalError)
	}

	return user.ID, nil
}

// ListUsers get all users from database.
func (s Repository) ListUsers(ctx context.Context, activeOnly bool) ([]model.User, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("pg.Repository: list users, active_only=%v", activeOnly)

	var users []UserDB

	sql := "SELECT id, username, password, email, name, created_at, updated_at FROM users"
	if activeOnly {
		sql += " WHERE password != 'LOCKED'"
	}

	sql += " ORDER BY username"

	dbctx := db.MustCtx(ctx)

	err := dbctx.SelectContext(ctx, &users, sql)
	if err != nil {
		return nil, aerr.Wrapf(err, "select users failed").WithTag(aerr.InternalError)
	}

	return usersFromDB(users), nil
}

// DeleteUser and all related objects.
func (s Repository) DeleteUser(ctx context.Context, userid int64) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msgf("pg.Repository: delete user user_id=%d", userid)

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(ctx,
		"DELETE FROM episodes WHERE podcast_id IN (SELECT id FROM podcasts WHERE user_id=$1)",
		userid)
	if err != nil {
		return aerr.Wrapf(err, "delete episodes failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM settings WHERE user_id=$1", userid); err != nil {
		return aerr.Wrapf(err, "delete settings failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM podcasts WHERE user_id=$1", userid); err != nil {
		return aerr.Wrapf(err, "delete podcasts failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM devices WHERE user_id=$1", userid); err != nil {
		return aerr.Wrapf(err, "delete devices failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	if _, err := dbctx.ExecContext(ctx, "DELETE FROM users WHERE id=$1", userid); err != nil {
		return aerr.Wrapf(err, "delete user failed").WithTag(aerr.InternalError).WithMeta("user_id", userid)
	}

	return nil
}
