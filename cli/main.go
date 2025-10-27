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
	"runtime/debug"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"gitlab.com/kabes/go-gpo/internal/cmd"

	_ "github.com/mattn/go-sqlite3"
)

var (
	Version   = "dev"
	Revision  = ""
	BuildDate = ""
	BuildUser = ""
	Branch    = ""
)

func buildVersionString() string {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			var dirty string

			for _, kv := range info.Settings {
				switch kv.Key {
				case "vcs.revision":
					Revision = kv.Value
				case "vcs.time":
					BuildDate = kv.Value
				case "vcs.modified":
					dirty = kv.Value
				}
			}

			return fmt.Sprintf("Rev: %s at %s %s", Revision, BuildDate, dirty)
		}
	} else {
		return fmt.Sprintf("Version: %s, Rev: %s, Build: %s by %s from %s",
			Version, Revision, BuildDate, BuildUser, Branch)
	}

	return Version
}

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "Print version.",
	}

	cmd := &cli.Command{
		Name:    "go-gpo",
		Version: buildVersionString(),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database", Value: "database.sqlite", Usage: "Database file"},
			&cli.StringFlag{Name: "log.level", Value: "info", Usage: "Log level (debug, info, warn, error)"},
			&cli.StringFlag{Name: "log.format", Value: "logfmt", Usage: "Log format (logfmt, json)"},
		},
		Commands: []*cli.Command{
			startServerCmd(),
			migrateCmd(),
			listCmd(),
			{
				Name:  "user",
				Usage: "manage users",
				Commands: []*cli.Command{
					addUserCmd(),
					changeUserPasswordCmd(),
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func startServerCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "start server",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "noauth", Value: false, Usage: "disable authentication"},
			&cli.StringFlag{Name: "address", Value: ":8080", Usage: "listen address"},
			&cli.BoolFlag{Name: "verbose", Value: false, Usage: "enable logging request and responses"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.Server{
				NoAuth:   c.Bool("noauth"),
				Database: c.String("database"),
				Listen:   c.String("address"),
				LogBody:  c.Bool("verbose"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func addUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "add new user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true},
			&cli.StringFlag{Name: "password", Required: true},
			&cli.StringFlag{Name: "email"},
			&cli.StringFlag{Name: "name"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.AddUser{
				Database: c.String("database"),
				Name:     c.String("name"),
				Password: c.String("password"),
				Email:    c.String("email"),
				Username: c.String("username"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func changeUserPasswordCmd() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "set new user password",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true},
			&cli.StringFlag{Name: "password", Required: true},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.ChangeUserPassword{
				Database: c.String("database"),
				Password: c.String("password"),
				Username: c.String("username"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func migrateCmd() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "update database",
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.Migrate{
				Database: c.String("database"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func listCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list user objects.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true},
			&cli.StringFlag{Name: "object", Required: true, Usage: "object to list (devices, subs)"},
			&cli.StringFlag{Name: "device"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.List{
				Database: c.String("database"),
				Username: c.String("username"),
				DeviceID: c.String("device"),
				Object:   c.String("object"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}
