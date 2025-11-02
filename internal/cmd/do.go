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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

const shutdownInjectorTimeout = 5 * time.Second

func createInjector(ctx context.Context) *do.RootScope {
	_ = ctx

	injector := do.New(
		service.Package,
		db.Package,
		repository.Package,
	)

	return injector
}

func explainDoInjecor(injector *do.RootScope) {
	explanation := do.ExplainInjector(injector)
	fmt.Println(explanation.String())
}

func shudownInjector(ctx context.Context, injector do.Injector) {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownInjectorTimeout)
	defer cancel()

	report := injector.RootScope().ShutdownWithContext(shutdownCtx)

	logger := log.Ctx(ctx)
	for k, err := range report.Errors {
		logger.Debug().Msgf("service shutdown error %v: %s", k, err)
	}
}
