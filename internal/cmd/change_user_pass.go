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

	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type ChangeUserPassword struct {
	Database string
	Password string
	Username string
}

func (a *ChangeUserPassword) Start(ctx context.Context) error {
	re := &repository.Database{}
	if err := re.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	user := model.User{
		Password: a.Password,
		Username: a.Username,
	}

	userv := service.NewUsersService(re)

	id, err := userv.ChangePassword(ctx, user)
	if err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	fmt.Printf("Changed password for user %q  id %d\n", a.Username, id)

	return nil
}
