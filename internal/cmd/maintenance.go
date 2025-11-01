//
// maintenance.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
)

type Maintenance struct {
	Database string
}

func (a *Maintenance) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	err := db.Maintenance(ctx)
	if err != nil {
		return fmt.Errorf("maintenance error: %w", err)
	}

	fmt.Printf("Done")

	return nil
}
