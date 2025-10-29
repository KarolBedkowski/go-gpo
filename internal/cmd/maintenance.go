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

	"gitlab.com/kabes/go-gpo/internal/db"
)

type Maintenance struct {
	Database string
}

func (a *Maintenance) Start(ctx context.Context) error {
	re := &db.Database{}
	if err := re.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	err := re.Maintenance(ctx)
	if err != nil {
		return fmt.Errorf("maintenance error: %w", err)
	}

	fmt.Printf("Done")

	return nil
}
