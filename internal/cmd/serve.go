//
// serve.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/server"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

type Server struct {
	Database      string
	Listen        string
	WebRoot       string
	DebugFlags    config.DebugFlags
	EnableMetrics bool
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting go-gpo (%s)...", config.VersionString)

	s.startSystemdWatchdog(logger)

	injector := s.createInjector(createInjector(ctx))

	if s.DebugFlags.HasFlag(config.DebugDo) {
		enableDoDebug(ctx, injector.RootScope())
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	database := do.MustInvoke[*db.Database](injector)
	if err := database.Connect(ctx, "sqlite3", s.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	database.RegisterMetrics(s.DebugFlags.HasFlag(config.DebugDBQueryMetrics))

	var group run.Group
	group.Add(run.ContextHandler(ctx))

	srv := do.MustInvoke[server.Server](injector)
	group.Add(func() error {
		logger.Log().Msgf("Listen on %s...", s.Listen)

		if err := srv.Start(ctx); err != nil {
			return fmt.Errorf("start server error: %w", err)
		}

		return nil
	}, srv.Stop)
	group.Add(func() error { return database.StartBackgroundMaintenance(ctx) }, func(_ error) {})

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

func (s *Server) createInjector(root do.Injector) *do.Scope {
	injector := root.Scope("server",
		gpoweb.Package,
		gpoapi.Package,
		server.Package,
	)

	do.ProvideNamedValue(injector, "server.webroot", s.WebRoot)
	do.ProvideValue(injector, &server.Configuration{
		Listen:        s.Listen,
		DebugFlags:    s.DebugFlags,
		WebRoot:       s.WebRoot,
		EnableMetrics: s.EnableMetrics,
	})

	return injector
}

func (s *Server) validate() error {
	s.Listen = strings.TrimSpace(s.Listen)
	s.WebRoot = strings.TrimSuffix(s.WebRoot, "/")

	if s.Listen == "" {
		return aerr.ErrValidation.WithUserMsg("listen address can't be empty")
	}

	return nil
}
