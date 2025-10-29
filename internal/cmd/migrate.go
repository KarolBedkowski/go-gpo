//
// migrate.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"embed"
	"fmt"

	"gitlab.com/kabes/go-gpo/internal/db"
)

//go:embed "migrations/*.sql"
var embedMigrations embed.FS

type Migrate struct {
	Database string
}

func (a *Migrate) Start(ctx context.Context) error {
	re := &db.Database{}
	if err := re.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	err := re.Migrate(ctx, "sqlite3", embedMigrations)
	if err != nil {
		return fmt.Errorf("migrate error: %w", err)
	}

	fmt.Printf("Migration finished")

	return nil
}
