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
	"gitlab.com/kabes/go-gpo/internal/db"
)

func wrap(
	cmdfunc func(ctx context.Context, clicmd *cli.Command, i do.Injector) error,
) func(ctx context.Context, clicmd *cli.Command) error {
	return func(ctx context.Context, clicmd *cli.Command) error {
		if err := initializeLogger(clicmd.String("log.level"), clicmd.String("log.format")); err != nil {
			return err
		}

		ctx = log.Logger.WithContext(ctx)

		dbconnstr := clicmd.String("db.connstr")
		if dbconnstr == "" {
			return aerr.New("db.connstr argument can't be empty").WithTag(aerr.ValidationError)
		}

		dbdriver := clicmd.String("db.driver")
		if dbdriver == "" {
			return aerr.New("db.driver argument can't be empty").WithTag(aerr.ValidationError)
		}

		// TODO: aliases, validation

		injector := createInjector(ctx)

		do.ProvideNamedValue(injector, "db.driver", clicmd.String("db.driver"))

		db := do.MustInvoke[*db.Database](injector)
		if err := db.Connect(ctx, dbdriver, dbconnstr); err != nil {
			return aerr.Wrapf(err, "connect to database failed")
		}

		defer shutdownInjector(ctx, injector)

		return cmdfunc(ctx, clicmd, injector)
	}
}
