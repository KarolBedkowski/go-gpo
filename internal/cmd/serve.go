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

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Server struct {
	NoAuth   bool
	Database string
	Listen   string
	LogBody  bool
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting server on %q...", s.Listen)

	re := &repository.Database{}
	if err := re.Connect(ctx, "sqlite3", s.Database+"?_fk=true"); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	cfg := api.Configuration{
		NoAuth:  s.NoAuth,
		Listen:  s.Listen,
		LogBody: s.LogBody,
	}

	if err := api.Start(ctx, re, &cfg); err != nil {
		return fmt.Errorf("start server error: %w", err)
	}

	return nil
}
