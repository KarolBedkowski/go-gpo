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

	//	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

type Settings struct {
	repo *repository.Repository
}

func NewSettingsService(repo *repository.Repository) *Settings {
	return &Settings{repo}
}

func (s Settings) GetSettings(ctx context.Context, username, scope, key string) (map[string]string, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	sett, err := tx.GetSettings(ctx, user.ID, scope, key)
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

func (e Settings) SaveSettings(
	ctx context.Context,
	username, scope, key string,
	set map[string]string,
	del []string,
) error {
	tx, err := e.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return ErrUnknownUser
	} else if err != nil {
		return fmt.Errorf("get user error: %w", err)
	}

	dbsett, err := tx.GetSettings(ctx, user.ID, scope, key)
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

	if err := tx.SaveSettings(ctx, &dbsett); err != nil {
		return fmt.Errorf("save settings error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}
