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

	"github.com/rs/zerolog/log"
)

func (t *Transaction) GetUser(ctx context.Context, username string) (UserDB, error) {
	user := UserDB{}

	err := t.tx.QueryRowxContext(ctx,
		"SELECT id, username, password, email, name, created_at, updated_at "+
			"FROM users WHERE username=?",
		username).
		StructScan(&user)

	switch {
	case err == nil:
		return user, nil
	case errors.Is(err, sql.ErrNoRows):
		return user, ErrNoData
	default:
		return user, fmt.Errorf("get user error: %w", err)
	}
}

func (t *Transaction) SaveUser(ctx context.Context, user *UserDB) (int64, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Object("user", user).Msg("save user")

	if user.ID == 0 {
		res, err := t.tx.ExecContext(ctx,
			"INSERT INTO users (username, password, email, name, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			user.Username, user.Password, user.Email, user.Name, user.CreatedAt, user.UpdatedAt)
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
	_, err := t.tx.ExecContext(ctx,
		"UPDATE users SET password=?, email=?, name=?, updated_at=? WHERE id=?",
		user.Password, user.Email, user.Name, user.UpdatedAt, user.ID)
	if err != nil {
		return user.ID, fmt.Errorf("update user error: %w", err)
	}

	return user.ID, nil
}
