// Package sqlite implement repository for database.
package sqlite

//
// sqlite.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/config"
)

type Repository struct{}

//------------------------------------------------------------------------------

const (
	ConnMaxIdleTime = 30 * time.Second
	ConnMaxLifetime = 60 * time.Second
	MaxIdleConns    = 1
	MaxOpenConns    = 10
)

//------------------------------------------------------------------------------

type Database struct {
	db      *sqlx.DB
	connstr string
}

func NewDatabaseI(i do.Injector) (*Database, error) {
	dbconf := do.MustInvoke[config.DBConfig](i)

	connstr, err := prepareSqliteConnstr(dbconf.Connstr)
	if err != nil {
		return nil, aerr.Wrapf(err, "invalid db.connstr")
	}

	return &Database{
		db:      nil,
		connstr: connstr,
	}, nil
}

func (d *Database) Open(ctx context.Context) (*sqlx.DB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("connecting to sqlite")

	var err error

	d.db, err = sqlx.Open("sqlite3", d.connstr)
	if err != nil {
		return nil, aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError).WithMeta("connstr", d.connstr)
	}

	d.db.SetConnMaxIdleTime(ConnMaxIdleTime)
	d.db.SetConnMaxLifetime(ConnMaxLifetime)
	d.db.SetMaxIdleConns(MaxIdleConns)
	d.db.SetMaxOpenConns(MaxOpenConns)

	if err := d.onOpenConn(ctx, d.db); err != nil {
		return nil, aerr.Wrapf(err, "open database failed - run init script error").
			WithTag(aerr.InternalError)
	}

	if err := d.db.PingContext(ctx); err != nil {
		return nil, aerr.Wrapf(err, "ping database failed").WithTag(aerr.InternalError)
	}

	return d.db, nil
}

func (d *Database) Shutdown(ctx context.Context) error {
	if d.db == nil {
		return nil
	}

	logger := log.Ctx(ctx)
	logger.Debug().Msg("closing database...")

	if err := d.db.Close(); err != nil {
		d.db = nil

		return aerr.Wrapf(err, "close db error")
	}

	d.db = nil

	return nil
}

func (d *Database) GetDB() *sql.DB {
	if d.db != nil {
		return d.db.DB
	}

	return nil
}

func (d *Database) GetConnection(ctx context.Context) (*sqlx.Conn, error) {
	conn, err := d.db.Connx(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	if err := d.onOpenConn(ctx, conn); err != nil {
		return nil, aerr.ApplyFor(aerr.ErrDatabase, err, "failed run onOpenConn scripts")
	}

	return conn, nil
}

func (d *Database) CloseConnection(ctx context.Context, conn *sqlx.Conn) error {
	if err := d.onCloseConn(ctx, conn); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "run scripts onClose failed")
	}

	if err := conn.Close(); err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "close connection failed")
	}

	return nil
}

func (d *Database) Migrate(ctx context.Context) error {
	logger := log.Ctx(ctx)

	migdir, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		panic(fmt.Errorf("prepare migration fs failed: %w", err))
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, d.db.DB, migdir)
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

	return nil
}

func (d *Database) Clear(ctx context.Context) error {
	sql := `
		PRAGMA foreign_keys=OFF;
		DELETE FROM settings;
		DELETE FROM episodes;
		DELETE FROM podcasts;
		DELETE FROM devices;
		DELETE FROM users;
		DELETE FROM sessions;
		PRAGMA foreign_keys=ON;
	`

	_, err := d.db.ExecContext(ctx, sql)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "clear database failed")
	}

	return nil
}

func (d *Database) HealthCheck(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return aerr.Wrapf(err, "ping database failed").WithTag(aerr.InternalError)
	}

	return nil
}

func (d *Database) onOpenConn(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		`PRAGMA temp_store = MEMORY;
		PRAGMA busy_timeout = 1000;
		`,
	)
	if err != nil {
		return aerr.Wrap(err)
	}

	return nil
}

func (d *Database) onCloseConn(ctx context.Context, db sqlx.ExecerContext) error {
	_, err := db.ExecContext(ctx,
		`PRAGMA optimize`,
	)
	if err != nil {
		return aerr.Wrap(err)
	}

	return nil
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

	if !query.Has("_journal_mode") {
		query.Set("_journal_mode", "WAL")
	}

	if !query.Has("_synchronous") {
		query.Set("_synchronous", "NORMAL")
	}

	parsed.RawQuery = query.Encode()

	return parsed.String(), err
}
