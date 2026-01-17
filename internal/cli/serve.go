package cli

//
// serve.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"context"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Merovius/systemd"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	gpoapi "gitlab.com/kabes/go-gpo/internal/api"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/config"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/server"
	"gitlab.com/kabes/go-gpo/internal/service"
	gpoweb "gitlab.com/kabes/go-gpo/internal/web"
)

func newStartServerCmd() *cli.Command { //nolint:funlen
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
			&cli.DurationFlag{
				Name:    "podcast-load-interval",
				Usage:   "Enable background worker that download podcast information in given intervals.",
				Sources: cli.EnvVars("GOGPO_SERVER_PODCAST_LOAD_INTERVAL"),
				Value:   0,
			},
			&cli.BoolFlag{
				Name:    "podcast-load-only-missing",
				Usage:   "Download podcast info only for podcasts without title (do not update).",
				Sources: cli.EnvVars("GOGPO_SERVER_PODCAST_LOAD_MISSING_ONLY"),
			},
			&cli.BoolFlag{
				Name:    "podcast-load-episodes",
				Usage:   "When loading podcast, load also episodes title.",
				Sources: cli.EnvVars("GOGPO_SERVER_PODCAST_LOAD_EPISODES"),
			},
			&cli.StringFlag{
				Name:    "mgmt-address",
				Value:   "",
				Usage:   "listen address for management endpoints; empty disable management; may be the same as main 'address'",
				Aliases: []string{"m"},
				Sources: cli.EnvVars("GOGPO_MGMT_SERVER_ADDRESS"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "mgmt-access-list",
				Value:   "",
				Usage:   "list of ip or networks separated by ',' allowed to connected to mgmt endpoints.",
				Sources: cli.EnvVars("GOGPO_MGMT_SERVER_ACCESS_LIST"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "session-store",
				Value:   "db",
				Usage:   "where store session data (db. memory)",
				Sources: cli.EnvVars("GOGPO_SESSION_STORE"),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.BoolFlag{
				Name:    "set-security-headers",
				Usage:   "enable add some http security related headers",
				Sources: cli.EnvVars("GOGPO_SET_SECURITY_HEADERS"),
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

	serverConf := config.ServerConf{
		MainServer: config.ListenConf{
			Address:      strings.TrimSpace(clicmd.String("address")),
			WebRoot:      strings.TrimSuffix(clicmd.String("web-root"), "/"),
			TLSKey:       clicmd.String("key"),
			TLSCert:      clicmd.String("cert"),
			CookieSecure: clicmd.Bool("secure-cookie"),
		},
		MgmtServer: config.ListenConf{
			Address: strings.TrimSpace(clicmd.String("mgmt-address")),
			// mgmt not use for now tls/webroot/cookie
		},
		DebugFlags:         config.NewDebugFLags(clicmd.String("debug")),
		EnableMetrics:      clicmd.Bool("enable-metrics"),
		MgmtAccessList:     clicmd.String("mgmt-access-list"),
		SessionStore:       clicmd.String("session-store"),
		SetSecurityHeaders: clicmd.Bool("set-security-headers"),
	}

	if err := serverConf.Validate(); err != nil {
		return aerr.Wrapf(err, "server config validation failed")
	}

	do.ProvideNamedValue(injector, "server.webroot", serverConf.MainServer.WebRoot)
	do.ProvideValue(injector, &serverConf)

	if serverConf.DebugFlags.HasFlag(config.DebugDo) {
		enableDoDebug(ctx, injector.RootScope())
	}

	s := Server{}

	return s.start(ctx, injector, &serverConf, clicmd)
}

type Server struct{}

func (s *Server) start(ctx context.Context, injector do.Injector, cfg *config.ServerConf,
	clicmd *cli.Command,
) error {
	logger := log.Ctx(ctx)
	logger.Log().Msgf("Starting go-gpo (%s)...", config.VersionString)
	logger.Debug().Msgf("Server: debug_flags=%q", cfg.DebugFlags)

	s.startSystemdWatchdog(logger)

	db.RegisterMetrics(injector, cfg.DebugFlags.HasFlag(config.DebugDBQueryMetrics))

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	srv := do.MustInvoke[*server.Server](injector)
	if err := srv.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msgf("start server failed error=%q", err)

		return aerr.New("failed start server")
	}

	if cfg.SeparateMgmtEnabled() {
		msrv := do.MustInvoke[*server.MgmtServer](injector)
		if err := msrv.Start(ctx); err != nil {
			logger.Fatal().Err(err).Msgf("start mgmt server failed error=%q", err)

			return aerr.New("failed start mgmt server")
		}
	}

	maintSrv := do.MustInvoke[*service.MaintenanceSrv](injector)
	go s.runBackgroundMaintenance(ctx, maintSrv)

	if i := clicmd.Duration("podcast-load-interval"); i > 0 {
		go s.podcastDownloadTask(ctx, injector, i, clicmd.Bool("podcast-load-episodes"),
			clicmd.Bool("podcast-load-only-missing"))
	}

	systemd.NotifyReady()           //nolint:errcheck
	systemd.NotifyStatus("running") //nolint:errcheck

	<-ctx.Done()

	systemd.NotifyStatus("stopped") //nolint:errcheck

	return nil
}

func (*Server) startSystemdWatchdog(logger *zerolog.Logger) {
	if ok, dur, err := systemd.AutoWatchdog(); ok {
		logger.Info().Msgf("Systemd: autowatchdog started; duration=%s", dur)
	} else if err != nil {
		logger.Warn().Err(err).Msgf("Systemd: autowatchdog start error=%q", err)
	}
}

func (s *Server) podcastDownloadTask(ctx context.Context, injector do.Injector,
	interval time.Duration, loadepisodes, missingonly bool,
) {
	logger := log.Ctx(ctx)
	logger.Info().Msgf("PodcastDownloader: start background podcast downloader; interval=%s", interval)

	podcastSrv := do.MustInvoke[*service.PodcastsSrv](injector)
	since := time.Now().Add(-24 * time.Hour).UTC()

	eventlog := common.NewEventLog("download podcast info", "worker")
	defer eventlog.Close()

	ctx = common.ContextWithEventLog(ctx, eventlog)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		start := time.Now()

		eventlog.Printf("start processing")

		if err := podcastSrv.DownloadPodcastsInfo(ctx, since, loadepisodes, missingonly); err != nil {
			logger.Error().Err(err).Msgf("PodcastDownloader: download podcast info job error=%q", err)
			eventlog.Errorf("processing error=%q", err)
		} else {
			eventlog.Printf("processing finished")
		}

		since = start
	}
}

func (s *Server) runBackgroundMaintenance(ctx context.Context, maintSrv *service.MaintenanceSrv) {
	const startHour = 4

	logger := log.Ctx(ctx)
	logger.Info().Msg("Maintenance: start background maintenance task")

	eventlog := common.NewEventLog("db maintenance", "worker")
	defer eventlog.Close()

	ctx = common.ContextWithEventLog(ctx, eventlog)

	for {
		now := time.Now().UTC()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), startHour, 0, 0, 0, time.UTC)

		if nextRun.Before(now) {
			nextRun = nextRun.Add(time.Duration(60*60*24) * time.Second) //nolint:mnd
		}

		wait := nextRun.Sub(now)

		logger.Debug().Msgf("Maintenance: next_run=%q wait=%q", nextRun, wait)
		eventlog.Printf("maintenance next_run=%q wait=%q", nextRun, wait)

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			taskid := xid.New()
			llog := logger.With().Str("task_id", taskid.String()).Logger() //nolint:nilaway
			eventlog.Printf("start maintenance task_id=%s", taskid.String())

			if err := maintSrv.MaintainDatabase(hlog.CtxWithID(ctx, taskid)); err != nil {
				llog.Error().Err(err).Msgf("Maintenance: run database maintenance task error=%q", err)
				eventlog.Errorf("maintenance error task_id=%s error=%q", taskid.String(), err)
			} else {
				eventlog.Printf("maintenance finished task_id=%s", taskid.String())
			}
		}
	}
}
