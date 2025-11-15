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
	logger := log.Ctx(ctx)
	logger.Debug().Msg("create injector")

	injector := do.NewWithOpts(
		&do.InjectorOpts{
			// Logf: func(format string, args ...any) {
			// 	logger.Debug().Msgf(format, args...)
			// },
		},
		service.Package,
		db.Package,
		repository.Package,
	)

	return injector
}

func enableDoDebug(ctx context.Context, injector *do.RootScope) {
	logger := log.Ctx(ctx)

	explanation := do.ExplainInjector(injector)
	fmt.Println(explanation.String())

	// injector.AddAfterInvocationHook(func(_ *do.Scope, serviceName string, err error) {
	// 	logger.Debug().Err(err).Msgf("service %q after invocation", serviceName)
	// })
	// injector.AddBeforeShutdownHook(func(_ *do.Scope, serviceName string) {
	// 	logger.Debug().Msgf("service %q start shutdown", serviceName)
	// })
	injector.AddAfterShutdownHook(func(_ *do.Scope, serviceName string, err error) {
		logger.Debug().Err(err).Msgf("service %q shutdown complete", serviceName)
	})
}

func shutdownInjector(ctx context.Context, injector do.Injector) {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownInjectorTimeout)
	defer cancel()

	report := injector.RootScope().ShutdownWithContext(shutdownCtx)

	logger := log.Ctx(ctx)
	for k, err := range report.Errors {
		logger.Debug().Msgf("service shutdown error %v: %s", k, err)
	}
}
