package main

//
// prom-logmonitor.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	// _ "github.com/WAY29/icecream-go/icecream".

	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	acli "gitlab.com/kabes/go-gpo/internal/cli"
	"gitlab.com/kabes/go-gpo/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "Print version.",
	}

	cli := &cli.Command{
		Name:    "go-gpo",
		Version: config.VersionString,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "database",
				Value:     "database.sqlite",
				Usage:     "Database file",
				Aliases:   []string{"D"},
				Sources:   cli.EnvVars("GOGPO_DB"),
				Validator: dbConnstrValidator,
				Config:    cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "log.level",
				Value:   "info",
				Usage:   "Log level (debug, info, warn, error)",
				Sources: cli.EnvVars("GOGPO_LOGLEVEL"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "log.format",
				Value:   "logfmt",
				Usage:   "Log format (logfmt, json, journald, syslog)",
				Sources: cli.EnvVars("GOGPO_LOGFORMAT"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{Name: "debug", Usage: "Debug flags", Sources: cli.EnvVars("GOGPO_DEBUG")},
		},
		Commands: []*cli.Command{
			acli.NewStartServerCmd(),
			acli.NewListCmd(),
			databaseSubCmd(),
			usersSubCmd(),
			devicesSubCmd(),
		},
	}

	if err := cli.Run(context.Background(), os.Args); err != nil {
		if h := aerr.GetUserMessage(err); h != "" {
			fmt.Printf("Error: %s\n", h)
		} else {
			fmt.Printf("Error: %s\n", err.Error())
		}

		if cli.String("log.level") == "debug" {
			fmt.Printf("Error: %#+v\n", err)
		}
	}
}

func usersSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "manage users",
		Commands: []*cli.Command{
			acli.NewAddUserCmd(),
			acli.NewDeleteUsersCmd(),
			acli.NewListUsersCmd(),
			acli.NewLockUserCmd(),
			acli.NewChangeUserPasswordCmd(),
		},
	}
}

func databaseSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "database",
		Usage: "manage database",
		Commands: []*cli.Command{
			acli.NewMigrateCmd(),
			acli.NewMaintenanceCmd(),
		},
	}
}

func devicesSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "device",
		Usage: "manage devices",
		Commands: []*cli.Command{
			acli.NewUpdateDeviceCmd(),
			acli.NewDeleteDeviceCmd(),
			acli.NewListDeviceCmd(),
		},
	}
}

//---------------------------------------------------------------------

func dbConnstrValidator(connstr string) error {
	if connstr == "" {
		return aerr.New("database connection string cannot be empty")
	}

	return nil
}
