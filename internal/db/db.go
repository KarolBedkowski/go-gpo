package db

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

//go:embed "migrations/*.sql"
var embedMigrations embed.FS

type Database struct {
	db *sqlx.DB

	queryDuration *prometheus.HistogramVec
}

func NewDatabaseI(_ do.Injector) (*Database, error) {
	return &Database{}, nil
}

func (r *Database) Connect(ctx context.Context, driver, connstr string) error {
	var err error

	// add some required parameters to connstr
	connstr, err = prepareSqliteConnstr(connstr)
	if err != nil {
		return err
	}

	logger := log.Ctx(ctx)
	logger.Info().Msgf("connecting to %q %q", driver, connstr)

	r.db, err = sqlx.Open(driver, connstr)
	if err != nil {
		return aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError).WithMeta("connstr", connstr)
	}

	r.db.SetConnMaxIdleTime(30 * time.Second) //nolint:mnd
	r.db.SetConnMaxLifetime(60 * time.Second) //nolint:mnd
	r.db.SetMaxIdleConns(1)
	r.db.SetMaxOpenConns(10) //nolint:mnd

	if err := r.onConnect(ctx, r.db); err != nil {
		return aerr.Wrapf(err, "call startup scripts error").WithTag(aerr.InternalError)
	}

	if err := r.db.PingContext(ctx); err != nil {
		return aerr.Wrapf(err, "ping database failed").WithTag(aerr.InternalError)
	}

	return nil
}

func (r *Database) RegisterMetrics(queryTime bool) {
	// gather stats from database
	prometheus.DefaultRegisterer.MustRegister(collectors.NewDBStatsCollector(r.db.DB, "main"))

	if queryTime {
		r.queryDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Tracks the latencies for database query.",
				Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"caller"},
		)

		prometheus.DefaultRegisterer.MustRegister(r.queryDuration)
	}
}

// Shutdown close database. Called by samber/do.
func (r *Database) Shutdown(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	if err := r.db.Close(); err != nil {
		return fmt.Errorf("close db error: %w", err)
	}

	logger := log.Ctx(ctx)
	logger.Debug().Msg("db closed")

	return nil
}

func (r *Database) Migrate(ctx context.Context, driver string) error { //nolint:cyclop
	if driver != "sqlite3" {
		panic("only sqlite3")
	}

	logger := log.Ctx(ctx)

	migdir, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		panic(fmt.Errorf("prepare migration fs failed: %w", err))
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, r.db.DB, migdir)
	if err != nil {
		panic(fmt.Errorf("create goose provider failed: %w", err))
	}

	ver, err := provider.GetDBVersion(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "failed to check current database version")
	}

	logger.Info().Msgf("current database version: %d", ver)

	for {
		res, err := provider.UpByOne(ctx)
		if res != nil {
			logger.Debug().Msgf("migration: %s", res)
		}

		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		} else if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "", "migrate database up failed")
		}
	}

	ver, err = provider.GetDBVersion(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "failed to check current database version")
	}

	logger.Info().Msgf("migrated database version: %d", ver)

	_, err = r.db.ExecContext(ctx, "PRAGMA optimize")
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute optimize script failed")
	}

	return nil
}

func (r *Database) GetConnection(ctx context.Context) (*sqlx.Conn, error) {
	conn, err := r.db.Connx(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	if err := r.onConnect(ctx, conn); err != nil {
		return nil, aerr.ApplyFor(aerr.ErrDatabase, err, "failed run onConnect scripts")
	}

	return conn, nil
}

func (r *Database) CloseConnection(ctx context.Context, conn *sqlx.Conn) {
	if err := r.onClose(ctx, conn); err != nil {
		log.Logger.Error().Err(err).Msg("run scripts onClose failed")
	}

	if err := conn.Close(); err != nil {
		log.Logger.Error().Err(err).Msg("close connection failed")
	}
}

func (r *Database) Maintenance(ctx context.Context) error {
	logger := log.Ctx(ctx)

	for idx, sql := range maintScripts {
		logger.Debug().Msgf("run maintenance script[%d]: %q", idx, sql)

		res, err := r.db.ExecContext(ctx, sql)
		if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance script failed").
				WithMeta("sql", sql)
		}

		rowsaffected, err := res.RowsAffected()
		if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance script - failed get rows affected").
				WithMeta("sql", sql)
		}

		logger.Debug().Msgf("run maintenance script[%d] finished; row affected: %d", idx, rowsaffected)
	}

	// print some stats
	var numEpisodes, numPodcasts int
	if err := r.db.GetContext(ctx, &numEpisodes, "SELECT count(*) FROM episodes"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count episodes failed")
	}

	if err := r.db.GetContext(ctx, &numPodcasts, "SELECT count(*) FROM podcasts"); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance - count podcasts failed")
	}

	logger.Info().Msgf("database maintenance finished; podcasts: %d; episodes: %d", numPodcasts, numEpisodes)

	return nil
}

func (r *Database) StartBackgroundMaintenance(ctx context.Context) error {
	const startHour = 4

	logger := log.Ctx(ctx)
	logger.Info().Msg("start background maintenance task")

	for {
		now := time.Now().UTC()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), startHour, 0, 0, 0, time.UTC)

		if nextRun.Before(now) {
			nextRun = nextRun.Add(time.Duration(60*60*24) * time.Second) //nolint:mnd
		}

		wait := nextRun.Sub(now)

		logger.Debug().Msgf("maintenance task - next run %s wait %s", nextRun, wait)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
			r.Maintenance(ctx)
		}
	}
}

func (r *Database) onConnect(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		"PRAGMA temp_store = MEMORY;",
	)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute onConnect script failed")
	}

	return nil
}

func (r *Database) onClose(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		"PRAGMA optimize",
	)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute onClose script failed")
	}

	return nil
}

func (r *Database) observeQueryDuration(start time.Time) {
	if r.queryDuration == nil {
		return
	}

	const skipFrames = 3

	rpc := make([]uintptr, 1)
	if n := runtime.Callers(skipFrames, rpc); n < 1 {
		return
	}

	frame, _ := runtime.CallersFrames(rpc).Next()
	if frame.PC == 0 {
		return
	}

	caller := frame.Function
	r.queryDuration.WithLabelValues(caller).Observe(time.Since(start).Seconds())
}

//------------------------------------------------------------------------------

func prepareSqliteConnstr(connstr string) (string, error) {
	if connstr == "" {
		return "", aerr.ErrInvalidConf.WithUserMsg("invalid (empty) database connection string")
	}

	if connstr == ":memory:" {
		return ":memory:?_fk=ON", nil
	}

	parsed, err := url.Parse(connstr)
	if err != nil {
		return "", aerr.ApplyFor(aerr.ErrInvalidConf, err, "", "failed to parse database connections string")
	}

	if parsed.Path == "" {
		return "", aerr.ErrInvalidConf.WithUserMsg("invalid database connection string - missing path")
	}

	query := parsed.Query()
	if !query.Has("_fk") && !query.Has("__foreign_keys") {
		query.Set("_fk", "ON")
	}

	parsed.RawQuery = query.Encode()

	return parsed.String(), err
}

//------------------------------------------------------------------------------

// InConnectionR run `fun` in database context. Open/close connection. Return `fun` result and error.
func InConnectionR[T any](ctx context.Context, r *Database,
	fun func(repository.DBContext) (T, error),
) (T, error) {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.GetConnection(ctx)
	if err != nil {
		return *new(T), err
	}

	defer r.CloseConnection(ctx, conn)

	res, err := fun(conn)
	if err != nil {
		return res, err
	}

	return res, nil
}

func InTransaction(ctx context.Context, r *Database, fun func(repository.DBContext) error) error {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.GetConnection(ctx)
	if err != nil {
		return err
	}

	defer r.CloseConnection(ctx, conn)

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	err = fun(tx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			merr := errors.Join(err, fmt.Errorf("commit error: %w", err))

			return aerr.ApplyFor(aerr.ErrDatabase, merr, "execute func in trans and rollback error")
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "commit tx failed")
	}

	return nil
}

// InTransactionR run `fun` in db transactions; return `fun` result and error.
func InTransactionR[T any](ctx context.Context, r *Database,
	fun func(repository.DBContext) (T, error),
) (T, error) {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.GetConnection(ctx)
	if err != nil {
		return *new(T), err
	}

	defer r.CloseConnection(ctx, conn)

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return *new(T), aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	res, err := fun(tx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			merr := errors.Join(err, fmt.Errorf("commit error: %w", err))

			return res, aerr.ApplyFor(aerr.ErrDatabase, merr, "execute func in trans and rollback error")
		}

		return res, err
	}

	if err := tx.Commit(); err != nil {
		return res, aerr.ApplyFor(aerr.ErrDatabase, err, "commit tx failed")
	}

	return res, nil
}

//------------------------------------------------------------------------------

var maintScripts = []string{
	// delete actions for episode if given episode has been deleted
	"DELETE FROM episodes AS e " +
		"WHERE action != 'delete' " +
		"AND updated_at < datetime('now','-1 month') " +
		"AND EXISTS (" +
		" SELECT NULL FROM episodes AS ed " +
		" WHERE ed.url = e.url AND ed.action = 'delete' AND ed.updated_at > e.updated_at);",
	// delete play actions when for given episode never play action exists
	"DELETE FROM episodes AS e " +
		"WHERE action = 'play' " +
		"AND updated_at < datetime('now','-14 day') " +
		"AND EXISTS (" +
		" SELECT NULL FROM episodes AS ed " +
		" WHERE ed.url = e.url AND ed.action = 'play' AND ed.updated_at > e.updated_at);",
	"VACUUM;",
	"ANALYZE;",
	"PRAGMA optimize;",
}

//------------------------------------------------------------------------------
