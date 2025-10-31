package cmd

//
// do.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func createInjector(ctx context.Context) do.Injector {
	injector := do.New(
		service.Package,
	)

	do.Provide(injector, db.NewDatabaseI)
	do.Provide(injector, repository.NewSqliteRepositoryI)

	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Available services: %v", injector.ListProvidedServices())

	explanation := do.ExplainInjector(injector)
	fmt.Println(explanation.String())

	return injector
}
