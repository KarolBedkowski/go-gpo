package cli

//
// migrate.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func newMigrateCmd() *cli.Command {
	return &cli.Command{
		Name:   "migrate",
		Usage:  "update database",
		Action: wrap(migrateCmd),
	}
}

func migrateCmd(ctx context.Context, _ *cli.Command, injector do.Injector) error {
	rdb := do.MustInvoke[repository.Database](injector)

	err := rdb.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("migrate error: %w", err)
	}

	//nolint:forbidigo
	fmt.Printf("Migration finished")

	return nil
}
