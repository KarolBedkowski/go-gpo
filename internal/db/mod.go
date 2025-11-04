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
	"net/url"
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

	logger := log.Ctx(ctx).With().Str("mod", "db").Logger()
	logger.Info().Msgf("connecting to %q %q", driver, connstr)

	r.db, err = sqlx.Open(driver, connstr)
	if err != nil {
		return aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError).WithMeta("connstr", connstr)
	}

	r.db.SetConnMaxIdleTime(30 * time.Second) //nolint:mnd
	r.db.SetConnMaxLifetime(60 * time.Second) //nolint:mnd
	r.db.SetMaxIdleConns(1)
	r.db.SetMaxOpenConns(10) //nolint:mnd

	// gather stats from database
	prometheus.DefaultRegisterer.MustRegister(collectors.NewDBStatsCollector(r.db.DB, "main"))

	if err := r.onConnect(ctx, r.db); err != nil {
		return aerr.Wrapf(err, "call startup scripts error").WithTag(aerr.InternalError)
	}

	if err := r.db.PingContext(ctx); err != nil {
		return aerr.Wrapf(err, "ping database failed").WithTag(aerr.InternalError)
	}

	return nil
}

func (r *Database) Shutdown(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	if err := r.db.Close(); err != nil {
		return fmt.Errorf("close db error: %w", err)
	}

	logger := log.Ctx(ctx)
	logger.Debug().Str("mod", "db").Msg("db closed")

	return nil
}

func (r *Database) Migrate(ctx context.Context, driver string) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect(driver); err != nil {
		panic(err)
	}

	if err := goose.UpContext(ctx, r.db.DB, "migrations"); err != nil {
		return fmt.Errorf("migrate up error: %w", err)
	}

	return nil
}

func (r *Database) GetConnection(ctx context.Context) (*sqlx.Conn, error) {
	conn, err := r.db.Connx(ctx)
	if err != nil {
		return nil, fmt.Errorf("open connection error: %w", err)
	}

	if err := r.onConnect(ctx, conn); err != nil {
		return nil, fmt.Errorf("on connect setup error: %w", err)
	}

	return conn, nil
}

func (r *Database) Begin(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx error: %w", err)
	}

	return tx, nil
}

func (r *Database) InTransaction(ctx context.Context, f func(repository.DBContext) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx error: %w", err)
	}

	err = f(tx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return errors.Join(err, fmt.Errorf("commit error: %w", err))
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

func (r *Database) Maintenance(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		"VACUUM;"+
			"PRAGMA optimize;",
	)
	if err != nil {
		return fmt.Errorf("execute db init script error: %w", err)
	}

	return nil
}

func (r *Database) onConnect(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		"PRAGMA temp_store = MEMORY;"+
			"PRAGMA optimize=0x10002;",
	)
	if err != nil {
		return fmt.Errorf("execute db init script error: %w", err)
	}

	return nil
}

func prepareSqliteConnstr(connstr string) (string, error) {
	parsed, err := url.Parse(connstr)
	if err != nil {
		return "", fmt.Errorf("failed to parse connections string: %w", err)
	}

	query := parsed.Query()
	if !query.Has("_fk") && !query.Has("__foreign_keys") {
		query.Set("_fk", "ON")
	}

	parsed.RawQuery = query.Encode()

	return parsed.String(), err
}
