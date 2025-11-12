//
// list_users.go
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
	"gitlab.com/kabes/go-gpo/internal/service"
)

type ListUsers struct {
	Database   string
	ActiveOnly bool
}

func (l *ListUsers) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", l.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	userv := do.MustInvoke[*service.UsersSrv](injector)

	users, err := userv.GetUsers(ctx, l.ActiveOnly)
	if err != nil {
		return fmt.Errorf("get users error: %w", err)
	}

	fmt.Printf("%-30s | %-30s | %-30s | %s \n", "User name", "Name", "Email", "Status")
	fmt.Println(
		"---------------------------------------------------------------------------------------------------------",
	)

	for _, u := range users {
		status := ""
		if u.Locked {
			status = "LOCKED"
		}

		fmt.Printf("%-30s | %-30s | %-30s | %s \n", u.Username, u.Name, u.Email, status)
	}

	return nil
}
