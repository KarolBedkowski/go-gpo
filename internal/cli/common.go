package cli

//
// common.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func wrap(
	cmdfunc func(ctx context.Context, clicmd *cli.Command, i do.Injector) error,
) func(ctx context.Context, clicmd *cli.Command) error {
	return func(ctx context.Context, clicmd *cli.Command) error {
		if err := initializeLogger(clicmd.String("log.level"), clicmd.String("log.format")); err != nil {
			return err
		}

		ctx = log.Logger.WithContext(ctx)

		dbconf := config.NewDBConfig(clicmd.String("db.driver"), clicmd.String("db.connstr"))

		if err := dbconf.Validate(); err != nil {
			return aerr.Wrapf(err, "invalid database configuration")
		}

		injector := createInjector(ctx)
		do.ProvideValue(injector, dbconf)

		db := do.MustInvoke[repository.Database](injector)
		if _, err := db.Open(ctx); err != nil {
			return aerr.Wrapf(err, "connect to database failed")
		}

		defer shutdownInjector(ctx, injector)

		return cmdfunc(ctx, clicmd, injector)
	}
}
