//
// maintenance.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func newMaintenanceCmd() *cli.Command {
	return &cli.Command{
		Name:   "maintenance",
		Usage:  "maintenance database",
		Action: wrap(maintenanceCmd),
	}
}

func maintenanceCmd(ctx context.Context, _ *cli.Command, injector do.Injector) error {
	db := do.MustInvoke[*db.Database](injector)
	repo := do.MustInvoke[repository.MaintenanceRepository](injector)

	err := db.Maintenance(ctx, repo)
	if err != nil {
		return fmt.Errorf("maintenance error: %w", err)
	}

	fmt.Printf("Done")

	return nil
}
