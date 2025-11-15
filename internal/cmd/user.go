//
// adduser.go
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

type AddUser struct {
	Database string
	Name     string
	Password string
	Email    string
	Username string
}

func (a *AddUser) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	usersrv := do.MustInvoke[*service.UsersSrv](injector)
	newuser := model.NewNewUser(a.Username, a.Password, a.Email, a.Name)

	id, err := usersrv.AddUser(ctx, &newuser)
	if err != nil {
		return fmt.Errorf("add user error: %w", err)
	}

	fmt.Printf("User %q created; id: %d\n", a.Username, id)

	return nil
}

// ---------------------------------------------------------------------

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

	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	users, err := usersrv.GetUsers(ctx, l.ActiveOnly)
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

// ---------------------------------------------------------------------

type DeleteUser struct {
	Database string
	Username string
}

func (d *DeleteUser) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", d.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	err := usersrv.DeleteUser(ctx, d.Username)
	if err != nil {
		return fmt.Errorf("delete user error: %w", err)
	}

	return nil
}
