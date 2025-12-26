//
// change_user_pass.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/service"
	"golang.org/x/term"
)

//---------------------------------------------------------------------

func newChangeUserPasswordCmd() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "set new user password / unlock account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "password", Aliases: []string{"p"}},
		},
		Action: wrap(changeUserPasswordCmd),
	}
}

func changeUserPasswordCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	pass, err := readValidatePassword(clicmd.String("password"))
	if err != nil {
		return err
	}

	username := clicmd.String("username")
	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	cmd := command.ChangeUserPasswordCmd{
		UserName:         username,
		Password:         pass,
		CurrentPassword:  "",
		CheckCurrentPass: false,
	}
	if err := usersrv.ChangePassword(ctx, &cmd); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	//nolint:forbidigo
	fmt.Printf("Changed password for user %q\n", username)

	return nil
}

func readValidatePassword(pass string) (string, error) {
	pass = strings.TrimSpace(pass)
	if pass == "" {
		//nolint:forbidigo
		fmt.Print("Enter new password: ")

		bytepw, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return "", fmt.Errorf("read password error: %w", err)
		}

		pass = strings.TrimSpace(string(bytepw))
	}

	if pass == "" {
		return "", errors.New("password can't be empty") //nolint:err113
	}

	return pass, nil
}

//---------------------------------------------------------------------

func newLockUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "lock user account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: wrap(lockUserCmd),
	}
}

func lockUserCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	username := clicmd.String("username")
	usersrv := do.MustInvoke[*service.UsersSrv](injector)

	la := command.LockAccountCmd{UserName: username}
	if err := usersrv.LockAccount(ctx, la); err != nil {
		return fmt.Errorf("change user password error: %w", err)
	}

	//nolint:forbidigo
	fmt.Printf("User %q locked\n", username)

	return nil
}
