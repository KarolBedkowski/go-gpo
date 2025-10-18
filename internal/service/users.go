//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

var ErrUnauthorized = errors.New("unauthorized")

type Users struct {
	repo *repository.Repository
}

func NewUsersService(repo *repository.Repository) *Users {
	return &Users{repo}
}

func (u *Users) LoginUser(ctx context.Context, username, password string) (*model.User, error) {
	user, err := u.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	if user == nil {
		return nil, ErrUnknownUser
	}

	if !user.CheckPassword(password) {
		return nil, ErrUnauthorized
	}

	return model.NewUserFromUserDB(user), nil
}
