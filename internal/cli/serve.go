//
// serve.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Merovius/systemd"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/server"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

func NewStartServerCmd() *cli.Command {
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
		Action: wrap(startServerCmd),
	}
}

func startServerCmd(ctx context.Context, clicmd *cli.Command, rootInjector do.Injector) error {
	injector := rootInjector.Scope("server",
		gpoweb.Package,
		gpoapi.Package,
		server.Package,
	)

	serverConf := server.Configuration{
		Listen:        strings.TrimSpace(clicmd.String("address")),
		DebugFlags:    config.NewDebugFLags(clicmd.String("debug")),
		WebRoot:       strings.TrimSuffix(clicmd.String("web-root"), "/"),
		EnableMetrics: clicmd.Bool("enable-metrics"),
	}

	if err := serverConf.Validate(); err != nil {
		return aerr.Wrapf(err, "server config validation failed")
	}

	do.ProvideNamedValue(injector, "server.webroot", clicmd.String("web-root"))
	do.ProvideValue(injector, &serverConf)

	if serverConf.DebugFlags.HasFlag(config.DebugDo) {
		enableDoDebug(ctx, injector.RootScope())
	}

	s := Server{
		db:   do.MustInvoke[*db.Database](injector),
		conf: serverConf,
	}

	return s.start(ctx, injector)
}

type Server struct {
	db   *db.Database
	conf server.Configuration
}

func (s *Server) start(ctx context.Context, injector do.Injector) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting go-gpo (%s)...", config.VersionString)

	s.startSystemdWatchdog(logger)
	s.db.RegisterMetrics(s.conf.DebugFlags.HasFlag(config.DebugDBQueryMetrics))

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	var group run.Group
	group.Add(run.ContextHandler(ctx))

	srv := do.MustInvoke[server.Server](injector)
	group.Add(func() error {
		logger.Log().Msgf("Listen on %s...", s.conf.Listen)

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("start server error: %w", err)
		}

		return nil
	}, srv.Stop)

	group.Add(func() error { return s.db.StartBackgroundMaintenance(ctx) }, func(_ error) {})

	systemd.NotifyReady()           //nolint:errcheck
	systemd.NotifyStatus("running") //nolint:errcheck

	if err := group.Run(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("start failed: %w", err)
	}

	shutdownInjector(ctx, injector)
	logger.Info().Msg("stopped")
	systemd.NotifyStatus("stopped") //nolint:errcheck

	return nil
}

func (*Server) startSystemdWatchdog(logger *zerolog.Logger) {
	if ok, dur, err := systemd.AutoWatchdog(); ok {
		logger.Info().Msgf("systemd autowatchdog started; duration=%s", dur)
	} else if err != nil {
		logger.Warn().Err(err).Msg("systemd autowatchdog start error")
	}
}
