//
// migrate.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
)

type Migrate struct {
	Database string
}

func (m *Migrate) Start(ctx context.Context) error {
	if err := m.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", m.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	err := db.Migrate(ctx, "sqlite3")
	if err != nil {
		return fmt.Errorf("migrate error: %w", err)
	}

	fmt.Printf("Migration finished")

	return nil
}

func (m *Migrate) validate() error {
	m.Database = strings.TrimSpace(m.Database)

	if m.Database == "" {
		return ErrValidation.Clone().WithUserMsg("database can't be empty")
	}

	return nil
}
