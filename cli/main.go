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
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/cmd"
	"gitlab.com/kabes/go-gpo/internal/config"

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
			&cli.StringFlag{
				Name:    "database",
				Value:   "database.sqlite",
				Usage:   "Database file",
				Aliases: []string{"d"},
				Sources: cli.EnvVars("GOGPO_DB"),
			},
			&cli.StringFlag{
				Name:    "log.level",
				Value:   "info",
				Usage:   "Log level (debug, info, warn, error)",
				Sources: cli.EnvVars("GOGPO_LOGLEVEL"),
			},
			&cli.StringFlag{
				Name:    "log.format",
				Value:   "logfmt",
				Usage:   "Log format (logfmt, json, syslog)",
				Sources: cli.EnvVars("GOGPO_LOGFORMAT"),
			},
			&cli.StringFlag{Name: "debug", Usage: "Debug flags", Sources: cli.EnvVars("GOGPO_DEBUG")},
		},
		Commands: []*cli.Command{
			startServerCmd(),
			listCmd(),
			databaseSubCmd(),
			usersSubCmd(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		if h := aerr.GetUserMessage(err); h != "" {
			fmt.Printf("Error: %s\n", h)
		} else {
			fmt.Printf("Error: %s\n", err.Error())
		}

		// TODO: verbose log
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
			},
			&cli.StringFlag{
				Name:    "web-root",
				Value:   "/",
				Usage:   "path root",
				Aliases: []string{"a"},
				Sources: cli.EnvVars("GOGPO_SERVER_WEBROOT"),
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
			listUsersCmd(),
			lockUserCmd(),
			changeUserPasswordCmd(),
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
				Username: c.String("username"),
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

func changeUserPasswordCmd() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "set new user password / unlock account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "password", Aliases: []string{"p"}},
		},
		Action: func(ctx context.Context, clicmd *cli.Command) error {
			initializeLogger(clicmd.String("log.level"), clicmd.String("log.format"))

			pass := strings.TrimSpace(clicmd.String("password"))
			if pass == "" {
				fmt.Print("Enter new password: ")

				bytepw, err := term.ReadPassword(syscall.Stdin)
				if err != nil {
					return fmt.Errorf("read password error: %w", err)
				}

				pass = strings.TrimSpace(string(bytepw))
			}

			if pass != "" {
				return errors.New("password can't be empty") //nolint:err113
			}

			s := cmd.ChangeUserPassword{
				Database: clicmd.String("database"),
				Password: pass,
				Username: clicmd.String("username"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}

func lockUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "lock user account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			initializeLogger(c.String("log.level"), c.String("log.format"))
			s := cmd.LockUserAccount{
				Database: c.String("database"),
				Username: c.String("username"),
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
				Database: c.String("database"),
				Username: c.String("username"),
				DeviceID: c.String("device"),
				Object:   c.String("object"),
			}

			return s.Start(log.Logger.WithContext(ctx))
		},
	}
}
