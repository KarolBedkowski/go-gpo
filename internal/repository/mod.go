package repository

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

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

var ErrNoData = errors.New("no result")

type Database struct {
	db *sqlx.DB
}

func (r *Database) Connect(ctx context.Context, driver, connstr string) error {
	var err error

	logger := log.Ctx(ctx)
	logger.Info().Msgf("connecting to %s/%s", connstr, driver)

	r.db, err = sqlx.Open(driver, connstr)
	if err != nil {
		return fmt.Errorf("open database error: %w", err)
	}

	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database error: %w", err)
	}

	return nil
}

func (r *Database) Migrate(ctx context.Context, driver string, em embed.FS) error {
	goose.SetBaseFS(em)

	if err := goose.SetDialect(driver); err != nil {
		panic(err)
	}

	if err := goose.UpContext(ctx, r.db.DB, "migrations"); err != nil {
		return fmt.Errorf("migrate up error:: %w", err)
	}

	return nil
}

func (r *Database) GetConnection(ctx context.Context) (*sqlx.Conn, error) {
	conn, err := r.db.Connx(ctx)
	if err != nil {
		return nil, fmt.Errorf("open connection error: %w", err)
	}

	return conn, nil
}

func (r *Database) Begin(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get transaction error: %w", err)
	}

	return tx, nil
}

func (r *Database) InTransaction(ctx context.Context, f func(DBContext) error) error {
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

func (r *Database) GetRepository(db DBContext) Repository {
	return sqliteRepository{db}
}
