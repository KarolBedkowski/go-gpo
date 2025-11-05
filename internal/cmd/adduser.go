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
	"strings"

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
	if err := a.validate(); err != nil {
		return err
	}

	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	userv := do.MustInvoke[*service.Users](injector)

	id, err := userv.AddUser(ctx, model.NewUser{
		Name:     a.Name,
		Password: a.Password,
		Email:    a.Email,
		Username: a.Username,
	})
	if err != nil {
		return fmt.Errorf("add user error: %w", err)
	}

	fmt.Printf("User %q created; id: %d\n", a.Username, id)

	return nil
}

func (a *AddUser) validate() error {
	a.Username = strings.TrimSpace(a.Username)
	a.Password = strings.TrimSpace(a.Password)
	a.Email = strings.TrimSpace(a.Email)
	a.Name = strings.TrimSpace(a.Name)

	if a.Username == "" {
		return ErrValidation.Clone().WithUserMsg("username can't be empty")
	}

	if a.Password == "" {
		return ErrValidation.Clone().WithUserMsg("password can't be empty")
	}

	return nil
}
