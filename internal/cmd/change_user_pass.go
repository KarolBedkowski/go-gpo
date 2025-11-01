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

type ChangeUserPassword struct {
	Database string
	Password string
	Username string
}

func (a *ChangeUserPassword) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	user := model.User{
		Password: a.Password,
		Username: a.Username,
	}

	userv := do.MustInvoke[*service.Users](injector)

	if err := userv.ChangePassword(ctx, user); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	fmt.Printf("Changed password for user %q\n", a.Username)

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

	if err := userv.LockAccount(ctx, l.Username); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	fmt.Printf("User %q locked\n", l.Username)

	return nil
}
