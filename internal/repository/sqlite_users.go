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
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func (s sqliteRepository) GetUser(ctx context.Context, username string) (UserDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Str("user_name", username).Msg("get user")

	user := UserDB{}

	err := s.db.GetContext(ctx, &user,
		"SELECT id, username, password, email, name, created_at, updated_at "+
			"FROM users WHERE username=?",
		username)

	switch {
	case err == nil:
		return user, nil
	case errors.Is(err, sql.ErrNoRows):
		return user, ErrNoData
	default:
		return user, fmt.Errorf("get user error: %w", err)
	}
}

func (s sqliteRepository) SaveUser(ctx context.Context, user *UserDB) (int64, error) {
	logger := log.Ctx(ctx)

	if user.ID == 0 {
		logger.Debug().Object("user", user).Msg("insert user")

		res, err := s.db.ExecContext(ctx,
			"INSERT INTO users (username, password, email, name, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			user.Username, user.Password, user.Email, user.Name, time.Now(), time.Now())
		if err != nil {
			return 0, fmt.Errorf("insert new user error: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id error: %w", err)
		}

		return id, nil
	}

	// update
	logger.Debug().Object("user", user).Msg("update user")

	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET password=?, email=?, name=?, updated_at=? WHERE id=?",
		user.Password, user.Email, user.Name, time.Now(), user.ID)
	if err != nil {
		return user.ID, fmt.Errorf("update user error: %w", err)
	}

	return user.ID, nil
}

// ListUsers get all users from database.
func (s sqliteRepository) ListUsers(ctx context.Context, activeOnly bool) ([]UserDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("list users, active_only=%v", activeOnly)

	var users []UserDB

	sql := "SELECT id, username, password, email, name, created_at, updated_at FROM users"
	if activeOnly {
		sql += " WHERE password != 'LOCKED'"
	}

	err := s.db.SelectContext(ctx, &users, sql)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	return users, nil
}
