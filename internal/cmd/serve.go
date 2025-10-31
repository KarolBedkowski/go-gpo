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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/server"
)

type Server struct {
	Database string
	Listen   string
	LogBody  bool
	WebRoot  string
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting server on %q...", s.Listen)

	if ok, dur, err := systemd.AutoWatchdog(); ok {
		logger.Info().Msgf("systemd autowatchdog started; duration=%s", dur)
	} else if err != nil {
		logger.Warn().Err(err).Msg("systemd autowatchdog start error")
	}

	re := &db.Database{}
	if err := re.Connect(ctx, "sqlite3", s.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	cfg := server.Configuration{
		Listen:  s.Listen,
		LogBody: s.LogBody,
		WebRoot: strings.TrimSuffix(s.WebRoot, "/"),
	}

	systemd.NotifyReady()           //nolint:errcheck
	systemd.NotifyStatus("running") //nolint:errcheck

	if err := server.Start(ctx, re, &cfg); err != nil {
		return fmt.Errorf("start server error: %w", err)
	}

	systemd.NotifyStatus("stopped") //nolint:errcheck

	return nil
}
