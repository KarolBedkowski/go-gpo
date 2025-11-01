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
	"time"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/server"
	"gitlab.com/kabes/go-gpo/internal/service"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

const shutdownInjectorTimeout = 5 * time.Second

func createInjector(ctx context.Context) *do.RootScope {
	injector := do.New(
		service.Package,
		db.Package,
		repository.Package,
		gpoweb.Package,
		gpoapi.Package,
		server.Package,
	)

	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Available services: %v", injector.ListProvidedServices())

	return injector
}

func explainDoInjecor(injector *do.RootScope) {
	explanation := do.ExplainInjector(injector)
	fmt.Println(explanation.String())
}

func shudownInjector(ctx context.Context, injector do.Injector) {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownInjectorTimeout)
	defer cancel()

	report := injector.ShutdownWithContext(shutdownCtx)

	logger := log.Ctx(ctx)
	for k, err := range report.Errors {
		logger.Debug().Msgf("Shutdown error %v: %s", k, err)
	}
}
