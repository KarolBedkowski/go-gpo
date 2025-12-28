package cli

//
// migexpimp.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

func newDataExportCmd() *cli.Command {
	return &cli.Command{
		Name:   "export",
		Usage:  "export data from database",
		Action: wrap(dataExportCmd),
	}
}

func dataExportCmd(ctx context.Context, _ *cli.Command, injector do.Injector) error {
	maintSrv := do.MustInvoke[*service.MaintenanceSrv](injector)

	data, err := maintSrv.ExportAll(ctx)
	if err != nil {
		return fmt.Errorf("export data error: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("export data error: %w", err)
	}

	return nil
}

//------------------------------------------------------------------------------

func newDataImportCmd() *cli.Command {
	return &cli.Command{
		Name:   "import",
		Usage:  "import data from database",
		Action: wrap(dataImportCmd),
	}
}

func dataImportCmd(ctx context.Context, _ *cli.Command, injector do.Injector) error {
	maintSrv := do.MustInvoke[*service.MaintenanceSrv](injector)

	var data []model.ExportStruct

	enc := json.NewDecoder(os.Stdin)
	if err := enc.Decode(&data); err != nil {
		return fmt.Errorf("parse data error: %w", err)
	}

	err := maintSrv.ImportAll(ctx, data)
	if err != nil {
		return fmt.Errorf("import data error: %w", err)
	}

	return nil
}
