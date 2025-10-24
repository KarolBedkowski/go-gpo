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

type Repository struct {
	db *sqlx.DB
}

func (r *Repository) Connect(ctx context.Context, driver, connstr string) error {
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

func (r *Repository) Begin(ctx context.Context) (Transaction, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin tx error: %w", err)
	}

	return Transaction{tx, false}, nil
}

func (r *Repository) Migrate(ctx context.Context, driver string, em embed.FS) error {
	goose.SetBaseFS(em)

	if err := goose.SetDialect(driver); err != nil {
		panic(err)
	}

	if err := goose.UpContext(ctx, r.db.DB, "migrations"); err != nil {
		return fmt.Errorf("migrate up error:: %w", err)
	}

	return nil
}

//---------------------------

type Transaction struct {
	tx        *sqlx.Tx
	committed bool
}

func (t *Transaction) Close() error {
	if !t.committed {
		if err := t.tx.Rollback(); err != nil {
			return fmt.Errorf("rollback error: %w", err)
		}
	}

	return nil
}

func (t *Transaction) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	t.committed = true

	return nil
}

//---------------------------

type queryer interface {
	sqlx.QueryerContext
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}
