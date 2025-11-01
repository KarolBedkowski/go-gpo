//
// serve.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Merovius/systemd"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/server"
)

type Server struct {
	Database   string
	Listen     string
	WebRoot    string
	DebugFlags config.DebugFlags
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting server on %q...", s.Listen)

	injector := createInjector(ctx)

	if s.DebugFlags.HasFlag("do") {
		explainDoInjecor(injector)
	}

	s.WebRoot = strings.TrimSuffix(s.WebRoot, "/")

	do.ProvideNamedValue(injector, "server.webroot", s.WebRoot)

	if ok, dur, err := systemd.AutoWatchdog(); ok {
		logger.Info().Msgf("systemd autowatchdog started; duration=%s", dur)
	} else if err != nil {
		logger.Warn().Err(err).Msg("systemd autowatchdog start error")
	}

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", s.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	cfg := server.Configuration{
		Listen:     s.Listen,
		DebugFlags: s.DebugFlags,
		WebRoot:    s.WebRoot,
	}

	systemd.NotifyReady()           //nolint:errcheck
	systemd.NotifyStatus("running") //nolint:errcheck

	if err := server.Start(ctx, injector, &cfg); err != nil {
		return fmt.Errorf("start server error: %w", err)
	}

	systemd.NotifyStatus("stopped") //nolint:errcheck

	return nil
}
