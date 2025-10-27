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

	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type AddUser struct {
	Database string
	Name     string
	Password string
	Email    string
	Username string
}

func (a *AddUser) Start(ctx context.Context) error {
	re := &repository.Database{}
	if err := re.Connect(ctx, "sqlite3", a.Database+"?_fk=true"); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	user := model.User{
		Name:     a.Name,
		Password: a.Password,
		Email:    a.Email,
		Username: a.Username,
	}

	userv := service.NewUsersService(re)

	id, err := userv.AddUser(ctx, user)
	if err != nil {
		return fmt.Errorf("add user error: %w", err)
	}

	fmt.Printf("User %q created; id: %d\n", a.Username, id)

	return nil
}
