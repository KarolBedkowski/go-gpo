package cmd

//
// common.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/db"
)

func wrap(
	cmdfunc func(ctx context.Context, clicmd *cli.Command, i do.Injector) error,
) func(ctx context.Context, clicmd *cli.Command) error {
	return func(ctx context.Context, clicmd *cli.Command) error {
		initializeLogger(clicmd.String("log.level"), clicmd.String("log.format"))

		ctx = log.Logger.WithContext(ctx)

		database := clicmd.String("database")
		if database == "" {
			return errors.New("database can't be empty") //nolint:err113
		}

		injector := createInjector(ctx)

		db := do.MustInvoke[*db.Database](injector)
		if err := db.Connect(ctx, "sqlite3", database); err != nil {
			return fmt.Errorf("connect to database error: %w", err)
		}

		defer db.Shutdown(ctx)

		return cmdfunc(ctx, clicmd, injector)
	}
}
