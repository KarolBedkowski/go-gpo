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

		dbconnstr := clicmd.String("db.connstr")
		if dbconnstr == "" {
			return aerr.New("db.connstr argument can't be empty").WithTag(aerr.ValidationError)
		}

		dbdriver, ok := validateDriverName(clicmd.String("db.driver"))
		if dbdriver == "" {
			return aerr.New("db.driver argument can't be empty").WithTag(aerr.ValidationError)
		} else if !ok {
			return aerr.New("invalid (unsupported) db.driver").WithTag(aerr.ValidationError)
		}

		injector := createInjector(ctx)

		do.ProvideNamedValue(injector, "db.driver", dbdriver)
		do.ProvideNamedValue(injector, "db.connstr", dbconnstr)

		db := do.MustInvoke[repository.Database](injector)
		if _, err := db.Open(ctx); err != nil {
			return aerr.Wrapf(err, "connect to database failed")
		}

		defer shutdownInjector(ctx, injector)

		return cmdfunc(ctx, clicmd, injector)
	}
}

func validateDriverName(driver string) (string, bool) {
	switch driver {
	case "sqlite", "sqlite3":
		return "sqlite3", true
	case "pg", "postgresql", "postgres":
		return "postgres", true
	}

	return driver, false
}
