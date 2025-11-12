//
// change_user_pass.go
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
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//---------------------------------------------------------------------

type ChangeUserPassword struct {
	Database string
	Password string
	Username string
}

func (c *ChangeUserPassword) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", c.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	userv := do.MustInvoke[*service.Users](injector)

	up := model.NewUserPassword(c.Username, c.Password)
	if err := userv.ChangePassword(ctx, &up); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	fmt.Printf("Changed password for user %q\n", c.Username)

	return nil
}

//---------------------------------------------------------------------

type LockUserAccount struct {
	Database string
	Username string
}

func (l *LockUserAccount) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", l.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	userv := do.MustInvoke[*service.Users](injector)

	la := model.NewLockAccount(l.Username)
	if err := userv.LockAccount(ctx, la); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	fmt.Printf("User %q locked\n", l.Username)

	return nil
}
