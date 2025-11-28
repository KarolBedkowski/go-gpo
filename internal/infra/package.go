package infra

//
// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/infra/sqlite"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var Package = do.Package(
	do.Lazy(func(_ do.Injector) (repository.SessionRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.UsersRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.PodcastsRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.DevicesRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.EpisodesRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.SettingsRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.MaintenanceRepository, error) {
		return &sqlite.SqliteRepository{}, nil
	}),
)
