package cmd

//
// do.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/samber/do"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func createInjector(ctx context.Context) *do.Injector {
	injector := do.New()

	do.Provide(injector, db.NewDatabaseI)
	do.Provide(injector, service.NewUsersServiceI)
	do.Provide(injector, service.NewDeviceServiceI)
	do.Provide(injector, service.NewEpisodesServiceI)
	do.Provide(injector, service.NewPodcastsServiceI)
	do.Provide(injector, service.NewSettingsServiceI)
	do.Provide(injector, service.NewSubssServiceI)

	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Available services: %v", injector.ListProvidedServices())

	return injector
}
