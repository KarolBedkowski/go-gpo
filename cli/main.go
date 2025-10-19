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

	"github.com/urfave/cli/v3"

	"gitlab.com/kabes/go-gpodder/internal/cmd"

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
		Name:    "go-gpodder",
		Version: buildVersionString(),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database", Value: "database.sqlite", Usage: "Database file"},
			&cli.StringFlag{Name: "log.level", Value: "info", Usage: "Log level (debug, info, warn, error)"},
			&cli.StringFlag{Name: "log.format", Value: "logfmt", Usage: "Log format (logfmt, json)"},
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "start server",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "noauth", Value: false, Usage: "disable authentication"},
					&cli.StringFlag{Name: "address", Value: "127.0.0.1:3000", Usage: "listen address"},
				},
				Action: startServerAction,
			},
			{
				Name:  "user",
				Usage: "manage users",
				Commands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add new user",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "username", Required: true},
							&cli.StringFlag{Name: "password", Required: true},
							&cli.StringFlag{Name: "email"},
							&cli.StringFlag{Name: "name"},
						},
						Action: addUserAction,
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func startServerAction(ctx context.Context, c *cli.Command) error {
	initializeLogger(c.String("log.level"), c.String("log.format"))
	s := cmd.Server{
		NoAuth:   c.Bool("noauth"),
		Database: c.String("database"),
		Listen:   c.String("address"),
	}

	return s.Start(ctx)
}

func addUserAction(ctx context.Context, c *cli.Command) error {
	initializeLogger(c.String("log.level"), c.String("log.format"))
	s := cmd.AddUser{
		Database: c.String("database"),
		Name:     c.String("name"),
		Password: c.String("password"),
		Email:    c.String("email"),
		Username: c.String("username"),
	}

	return s.Start(ctx)
}
