//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"encoding/json"
	"errors"
	"maps"

	//	"gitlab.com/kabes/go-gpo/internal/model"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Settings struct {
	db        *db.Database
	settRepo  repository.SettingsRepository
	usersRepo repository.UsersRepository
}

func NewSettingsServiceI(i do.Injector) (*Settings, error) {
	return &Settings{
		db:        do.MustInvoke[*db.Database](i),
		settRepo:  do.MustInvoke[repository.SettingsRepository](i),
		usersRepo: do.MustInvoke[repository.UsersRepository](i),
	}, nil
}

func (s Settings) GetSettings(ctx context.Context, username, scope, key string) (map[string]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := s.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	sett, err := s.settRepo.GetSettings(ctx, conn, user.ID, scope, key)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	settings := make(map[string]string)

	if len(sett.Value) == 0 {
		return settings, nil
	}

	if err := json.Unmarshal([]byte(sett.Value), &settings); err != nil {
		return nil, aerr.Wrapf(err, "failed unmarshal settings from database").WithMeta("value", sett.Value)
	}

	return settings, nil
}

func (s Settings) SaveSettings(
	ctx context.Context,
	username, scope, key string,
	set map[string]string,
	del []string,
) error {
	err := s.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		user, err := s.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		dbsett, err := s.settRepo.GetSettings(ctx, dbctx, user.ID, scope, key)
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		settings := make(map[string]string)

		if len(dbsett.Value) > 0 {
			if err := json.Unmarshal([]byte(dbsett.Value), &settings); err != nil {
				return aerr.Wrapf(err, "failed unmarshal settings from database").WithMeta("value", dbsett.Value)
			}
		}

		maps.Copy(settings, set)

		for _, k := range del {
			delete(settings, k)
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return aerr.Wrapf(err, "failed marshal settings").WithMeta("value", settings)
		}

		dbsett.Value = string(data)

		if err := s.settRepo.SaveSettings(ctx, dbctx, &dbsett); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})

	return err //nolint:wrapcheck
}
