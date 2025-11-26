//
// serve.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Merovius/systemd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/server"
	"gitlab.com/kabes/go-gpo/internal/service"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

func newStartServerCmd() *cli.Command {
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
			&cli.StringFlag{
				Name:      "cert",
				Usage:     "tls certificate file",
				Sources:   cli.EnvVars("GOGPO_SERVER_CERT"),
				Config:    cli.StringConfig{TrimSpace: true},
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "key",
				Usage:     "tls key file",
				Sources:   cli.EnvVars("GOGPO_SERVER_KEY"),
				Config:    cli.StringConfig{TrimSpace: true},
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:    "secure-cookie",
				Usage:   "use secure (https only) cookie",
				Sources: cli.EnvVars("GOGPO_SERVER_SECURE_COOKIE"),
			},
			&cli.BoolFlag{
				Name:    "enable-podcasts-loader",
				Usage:   "Enable background worker that download podcast information. This may east a lot of memory....",
				Sources: cli.EnvVars("GOGPO_SERVER_PODCAST_LOADER"),
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
		TLSKey:        clicmd.String("key"),
		TLSCert:       clicmd.String("cert"),
		CookieSecure:  clicmd.Bool("secure-cookie"),
	}

	if err := serverConf.Validate(); err != nil {
		return aerr.Wrapf(err, "server config validation failed")
	}

	do.ProvideNamedValue(injector, "server.webroot", serverConf.WebRoot)
	do.ProvideValue(injector, &serverConf)

	if serverConf.DebugFlags.HasFlag(config.DebugDo) {
		enableDoDebug(ctx, injector.RootScope())
	}

	s := Server{}

	return s.start(ctx, injector, &serverConf, clicmd.Bool("enable-podcasts-loader"))
}

type Server struct{}

func (s *Server) start(ctx context.Context, injector do.Injector, cfg *server.Configuration, podcastWorker bool) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting go-gpo (%s)...", config.VersionString)

	s.startSystemdWatchdog(logger)

	database := do.MustInvoke[*db.Database](injector)
	database.RegisterMetrics(cfg.DebugFlags.HasFlag(config.DebugDBQueryMetrics))

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	srv := do.MustInvoke[server.Server](injector)
	if err := srv.Start(ctx); err != nil {
		return aerr.Wrapf(err, "start server failed")
	}

	maintRepo := do.MustInvoke[repository.MaintenanceRepository](injector)

	go database.RunBackgroundMaintenance(ctx, maintRepo)

	if podcastWorker {
		go s.backgroundWorker(ctx, injector)
	}

	systemd.NotifyReady()           //nolint:errcheck
	systemd.NotifyStatus("running") //nolint:errcheck

	<-ctx.Done()

	systemd.NotifyStatus("stopped") //nolint:errcheck
	logger.Info().Msg("server stopped")

	return nil
}

func (*Server) startSystemdWatchdog(logger *zerolog.Logger) {
	if ok, dur, err := systemd.AutoWatchdog(); ok {
		logger.Info().Msgf("systemd autowatchdog started; duration=%s", dur)
	} else if err != nil {
		logger.Warn().Err(err).Msg("systemd autowatchdog start error")
	}
}

func (s *Server) backgroundWorker(ctx context.Context, injector do.Injector) {
	logger := log.Ctx(ctx)
	logger.Info().Msg("start background worker")

	podcastSrv := do.MustInvoke[*service.PodcastsSrv](injector)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Hour):
		}

		since := time.Now().Add(-24 * time.Hour).UTC()

		if err := podcastSrv.DownloadPodcastsInfo(ctx, since); err != nil {
			logger.Error().Err(err).Msg("download podcast info failed")
		}
	}
}
