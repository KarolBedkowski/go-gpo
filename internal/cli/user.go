package cli

//
// user.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//---------------------------------------------------------------------

func newAddUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "add new user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "password", Required: true, Aliases: []string{"p"}},
			&cli.StringFlag{Name: "email", Aliases: []string{"e"}},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
		},
		Action: wrap(addUserCmd),
	}
}

//nolint:forbidigo
func addUserCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	username := clicmd.String("username")

	usersrv := do.MustInvoke[*service.UsersSrv](injector)
	cmd := command.NewUserCmd{
		UserName: username,
		Password: clicmd.String("password"),
		Email:    clicmd.String("email"),
		Name:     clicmd.String("name"),
	}

	res, err := usersrv.AddUser(ctx, &cmd)
	switch {
	case err != nil:
		return fmt.Errorf("add user error: %w", err)
	case res.UserID > 0:
		fmt.Printf("User %q created; id: %d\n", username, res.UserID)
	default:
		fmt.Printf("Create user failed\n")
	}

	return nil
}

// ---------------------------------------------------------------------

func newListUsersCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list user accounts",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "active-only", Usage: "show active only accounts", Aliases: []string{"a"}},
		},
		Action: wrap(listUsersCmd),
	}
}

//nolint:forbidigo
func listUsersCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	users, err := usersrv.GetUsers(ctx, clicmd.Bool("active-only"))
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

		fmt.Printf("%-30s | %-30s | %-30s | %s \n", u.UserName, u.Name, u.Email, status)
	}

	return nil
}

// ---------------------------------------------------------------------

func newDeleteUsersCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete user account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: wrap(deleteUserCmd),
	}
}

func deleteUserCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	username := clicmd.String("username")
	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	err := usersrv.DeleteUser(ctx, &command.DeleteUserCmd{UserName: username})
	if err != nil {
		return fmt.Errorf("delete user error: %w", err)
	}

	//nolint:forbidigo
	fmt.Printf("User %s deleted\n", username)

	return nil
}
