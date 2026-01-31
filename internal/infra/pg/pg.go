// Package pg implement repositories for PsotgreSQL database.
package pg

//
// pg.go
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
	ConnMaxIdleTime = 300 * time.Second
	ConnMaxLifetime = 600 * time.Second
	MaxIdleConns    = 1
	MaxOpenConns    = 10
)

// ------------------------------------------------------------------------------

type Database struct {
	db      *sqlx.DB
	connstr string
}

func NewDatabaseI(i do.Injector) (*Database, error) {
	dbconf := do.MustInvoke[config.DBConfig](i)

	return &Database{
		db:      nil,
		connstr: dbconf.Connstr,
	}, nil
}

func (d *Database) Open(ctx context.Context) (*sqlx.DB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("pg.Database:: connecting to postgresql")

	var err error

	d.db, err = sqlx.Open("pgx", d.connstr)
	if err != nil {
		return nil, aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError).
			WithMeta("connstr", d.connstr)
	}

	d.db.SetConnMaxIdleTime(ConnMaxIdleTime)
	d.db.SetConnMaxLifetime(ConnMaxLifetime)
	d.db.SetMaxIdleConns(MaxIdleConns)
	d.db.SetMaxOpenConns(MaxOpenConns)

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
	logger.Debug().Msg("pg.Database:: closing database...")

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

	return conn, nil
}

func (d *Database) CloseConnection(ctx context.Context, conn *sqlx.Conn) error {
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

	provider, err := goose.NewProvider(goose.DialectPostgres, d.db.DB, migdir)
	if err != nil {
		panic(fmt.Errorf("create goose provider failed: %w", err))
	}

	ver, err := provider.GetDBVersion(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "", "failed to check current database version")
	}

	logger.Info().Msgf("pg.Database:: current database version: %d", ver)

	for {
		res, err := provider.UpByOne(ctx)
		if res != nil {
			logger.Debug().Msgf("pg.Database:: migration: %s", res)
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

	logger.Info().Msgf("pg.Database: migrated database version: %d", ver)

	return nil
}

func (d *Database) Clear(ctx context.Context) error {
	sqls := []string{
		"DELETE FROM settings;",
		"DELETE FROM episodes_hist;",
		"DELETE FROM episodes;",
		"DELETE FROM podcasts;",
		"DELETE FROM devices;",
		"DELETE FROM users;",
		"DELETE FROM sessions;",
	}

	for _, sql := range sqls {
		_, err := d.db.ExecContext(ctx, sql)
		if err != nil {
			return aerr.ApplyFor(aerr.ErrDatabase, err, "clear database failed").WithMeta("sql", sql)
		}
	}

	return nil
}

func (d *Database) HealthCheck(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return aerr.Wrapf(err, "ping database failed").WithTag(aerr.InternalError)
	}

	return nil
}
