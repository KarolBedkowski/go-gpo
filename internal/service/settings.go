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
	"fmt"
	"maps"

	//	"gitlab.com/kabes/go-gpo/internal/model"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Settings struct {
	db *db.Database
}

func NewSettingsService(db *db.Database) *Settings {
	return &Settings{db}
}

func NewSettingsServiceI(i do.Injector) (*Settings, error) {
	db := do.MustInvoke[*db.Database](i)

	return &Settings{db}, nil
}

func (s Settings) GetSettings(ctx context.Context, username, scope, key string) (map[string]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := s.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	sett, err := repo.GetSettings(ctx, user.ID, scope, key)
	if err != nil {
		return nil, fmt.Errorf("get settings error: %w", err)
	}

	settings := make(map[string]string)

	if len(sett.Value) == 0 {
		return settings, nil
	}

	if err := json.Unmarshal([]byte(sett.Value), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal settings error: %w", err)
	}

	return settings, nil
}

func (s Settings) SaveSettings(
	ctx context.Context,
	username, scope, key string,
	set map[string]string,
	del []string,
) error {
	err := s.db.InTransaction(ctx, func(db repository.DBContext) error {
		repo := s.db.GetRepository(db)

		user, err := repo.GetUser(ctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return fmt.Errorf("get user error: %w", err)
		}

		dbsett, err := repo.GetSettings(ctx, user.ID, scope, key)
		if err != nil {
			return fmt.Errorf("get settings error: %w", err)
		}

		settings := make(map[string]string)

		if len(dbsett.Value) > 0 {
			if err := json.Unmarshal([]byte(dbsett.Value), &settings); err != nil {
				return fmt.Errorf("unmarshal settings error: %w", err)
			}
		}

		maps.Copy(settings, set)

		for _, k := range del {
			delete(settings, k)
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return fmt.Errorf("marshal settings error: %w", err)
		}

		dbsett.Value = string(data)

		if err := repo.SaveSettings(ctx, &dbsett); err != nil {
			return fmt.Errorf("save settings error: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("save settings error: %w", err)
	}

	return nil
}
