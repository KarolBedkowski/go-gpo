package pg

//
// pg.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

type Repository struct{}

type Database struct {
	db      *sqlx.DB
	connstr string
}

func NewDatabaseI(i do.Injector) (*Database, error) {
	connstr, err := do.InvokeNamed[string](i, "db.connstr")
	if err != nil {
		return nil, aerr.Wrapf(err, "invoke db.connstr failed").WithTag(aerr.InternalError)
	}

	if connstr == "" {
		return nil, aerr.Wrapf(err, "empty db.connstr")
	}

	return &Database{
		db:      nil,
		connstr: connstr,
	}, nil
}

func (d *Database) Open(ctx context.Context) (*sqlx.DB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("connecting to %q", d.connstr)

	var err error

	d.db, err = sqlx.Open("pgx", d.connstr)
	if err != nil {
		return nil, aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError).WithMeta("connstr", d.connstr)
	}

	d.db.SetConnMaxIdleTime(30 * time.Second) //nolint:mnd
	d.db.SetConnMaxLifetime(60 * time.Second) //nolint:mnd
	d.db.SetMaxIdleConns(1)
	d.db.SetMaxOpenConns(10) //nolint:mnd

	return d.db, nil
}

func (d *Database) Close(ctx context.Context) error {
	if d.db == nil {
		return nil
	}

	if err := d.db.Close(); err != nil {
		d.db = nil

		return aerr.Wrapf(err, "close db error")
	}

	d.db = nil

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

func (*Database) OnOpenConn(ctx context.Context, db sqlx.ExecerContext) error {
	return nil
}

func (*Database) OnCloseConn(ctx context.Context, db sqlx.ExecerContext) error {
	return nil
}

func (d *Database) Clear(ctx context.Context) error {
	sqls := []string{
		"DELETE FROM settings;",
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
