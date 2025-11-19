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

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/cmd"
	"gitlab.com/kabes/go-gpo/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "Print version.",
	}

	cmd := &cli.Command{
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
			startServerCmd(),
			listCmd(),
			databaseSubCmd(),
			usersSubCmd(),
			devicesSubCmd(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		if h := aerr.GetUserMessage(err); h != "" {
			fmt.Printf("Error: %s\n", h)
		} else {
			fmt.Printf("Error: %s\n", err.Error())
		}

		if cmd.String("log.level") == "debug" {
			fmt.Printf("Error: %#+v\n", err)
		}
	}
}

func startServerCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "start server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Value:   ":8080",
				Usage:   "listen address",
				Aliases: []string{"a"},
				Sources: cli.EnvVars("GOGPO_SERVER_ADDRESS"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "web-root",
				Value:   "/",
				Usage:   "path root",
				Aliases: []string{"a"},
				Sources: cli.EnvVars("GOGPO_SERVER_WEBROOT"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.BoolFlag{
				Name:    "enable-metrics",
				Usage:   "enable prometheus metrics (/metrics endpoint)",
				Sources: cli.EnvVars("GOGPO_SERVER_METRICS"),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.Server{
				Database:      c.String("database"),
				Listen:        c.String("address"),
				DebugFlags:    config.NewDebugFLags(c.String("debug")),
				WebRoot:       c.String("web-root"),
				EnableMetrics: c.Bool("enable-metrics"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func usersSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "manage users",
		Commands: []*cli.Command{
			addUserCmd(),
			deleteUsersCmd(),
			listUsersCmd(),
			cmd.NewLockUserCmd(),
			cmd.NewChangeUserPasswordCmd(),
		},
	}
}

func addUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "add new user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "password", Required: true, Aliases: []string{"p"}},
			&cli.StringFlag{Name: "email", Aliases: []string{"e"}},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.AddUser{
				Database: c.String("database"),
				Name:     c.String("name"),
				Password: c.String("password"),
				Email:    c.String("email"),
				UserName: c.String("username"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func deleteUsersCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete user account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.DeleteUser{
				Database: c.String("database"),
				UserName: c.String("username"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func listUsersCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list user accounts",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "active-only", Usage: "show active only accounts", Aliases: []string{"a"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.ListUsers{
				Database:   c.String("database"),
				ActiveOnly: c.Bool("active-only"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func databaseSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "database",
		Usage: "manage database",
		Commands: []*cli.Command{
			migrateCmd(),
			maintenanceCmd(),
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

func maintenanceCmd() *cli.Command {
	return &cli.Command{
		Name:  "maintenance",
		Usage: "maintenance database",
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.Maintenance{
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
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{
				Name:     "object",
				Required: true,
				Usage:    "object to list (" + cmd.ListSupportedObjects + ")",
				Aliases:  []string{"o"},
			},
			&cli.StringFlag{Name: "device", Aliases: []string{"d"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.List{
				Database:   c.String("database"),
				UserName:   c.String("username"),
				DeviceName: c.String("device"),
				Object:     c.String("object"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func dbConnstrValidator(connstr string) error {
	if connstr == "" {
		return aerr.New("database connection string cannot be empty")
	}

	return nil
}

func devicesSubCmd() *cli.Command {
	return &cli.Command{
		Name:  "device",
		Usage: "manage devices",
		Commands: []*cli.Command{
			updateDeviceCmd(),
			deleteDeviceCmd(),
			listDeviceCmd(),
		},
	}
}

func updateDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "add or update device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "device", Required: true, Aliases: []string{"d"}},
			&cli.StringFlag{
				Name: "type", Required: false, Aliases: []string{"t"}, Value: "mobile",
				Usage: "device type (desktop, laptop, mobile, server, other)",
			},
			&cli.StringFlag{Name: "caption", Required: false, Aliases: []string{"c"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.UpdateDevice{
				Database:      c.String("database"),
				UserName:      c.String("username"),
				DeviceName:    c.String("device"),
				DeviceType:    c.String("type"),
				DeviceCaption: c.String("caption"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func deleteDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "device", Required: true, Aliases: []string{"d"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.DeleteDevice{
				Database:   c.String("database"),
				UserName:   c.String("username"),
				DeviceName: c.String("device"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func listDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))

			s := cmd.List{
				Database:   c.String("database"),
				UserName:   c.String("username"),
				DeviceName: c.String("device"),
				Object:     "devices",
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}
