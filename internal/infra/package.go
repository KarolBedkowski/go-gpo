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
	do.Lazy(func(_ do.Injector) (repository.Sessions, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Users, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Podcasts, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Devices, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Episodes, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Settings, error) {
		return &sqlite.Repository{}, nil
	}),
	do.Lazy(func(_ do.Injector) (repository.Maintenance, error) {
		return &sqlite.Repository{}, nil
	}),
)
