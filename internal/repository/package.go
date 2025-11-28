package repository

//
// package.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(func(_ do.Injector) (SessionRepository, error) {
		return &SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (UsersRepository, error) {
		return &SqliteRepository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (PodcastsRepository, error) {
		return &SqliteRepository{}, nil
	}),
	// do.Lazy(func(_ do.Injector) (DevicesRepository, error) {
	// 	return &SqliteRepository{}, nil
	// }),
	do.Lazy(func(_ do.Injector) (EpisodesRepository, error) {
		return &SqliteRepository{}, nil
	}),
	// do.Lazy(func(_ do.Injector) (SettingsRepository, error) {
	// 	return &SqliteRepository{}, nil
	// }),
	do.Lazy(func(_ do.Injector) (MaintenanceRepository, error) {
		return &SqliteRepository{}, nil
	}),
)
