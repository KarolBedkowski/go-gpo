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
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "migrate database up failed")
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

func (r *Database) InTransaction(ctx context.Context, fun func(repository.DBContext) error) error {
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

func (r *Database) Maintenance(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		"VACUUM;"+
			"PRAGMA optimize;",
	)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "execute maintenance script failed")
	}

	return nil
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

func prepareSqliteConnstr(connstr string) (string, error) {
	if connstr == "" {
		return "", aerr.ErrInvalidConf.WithUserMsg("invalid (empty) database connection string")
	}

	parsed, err := url.Parse(connstr)
	if err != nil {
		return "", aerr.ApplyFor(aerr.ErrInvalidConf, err, "", "failed to parse database connections string")
	}

	if parsed.Path == "" {
		return "", aerr.ApplyFor(aerr.ErrInvalidConf, err, "", "invalid database connection string - missing path")
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

// InTransactionR run `fun` in db transactions; return `fun` result and error.
func InTransactionR[T any](ctx context.Context, r *Database,
	fun func(repository.DBContext) (T, error),
) (T, error) {
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
