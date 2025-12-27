package cli

//
// maintenance.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func newMaintenanceCmd() *cli.Command {
	return &cli.Command{
		Name:   "maintenance",
		Usage:  "maintenance database",
		Action: wrap(maintenanceCmd),
	}
}

func maintenanceCmd(ctx context.Context, _ *cli.Command, injector do.Injector) error {
	maintSrv := do.MustInvoke[*service.MaintenanceSrv](injector)

	err := maintSrv.MaintainDatabase(ctx)
	if err != nil {
		return fmt.Errorf("maintenance error: %w", err)
	}

	//nolint:forbidigo
	fmt.Printf("Done")

	return nil
}
