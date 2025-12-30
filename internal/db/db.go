package db

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Database struct {
	dbimpl repository.Database

	queryDuration *prometheus.HistogramVec
}

func NewDatabaseI(i do.Injector) (*Database, error) {
	return &Database{
		dbimpl: do.MustInvoke[repository.Database](i),
	}, nil
}

func (r *Database) Connect(ctx context.Context) error {
	var err error

	logger := log.Ctx(ctx)
	logger.Info().Msg("connecting to database")

	_, err = r.dbimpl.Open(ctx)
	if err != nil {
		return aerr.Wrapf(err, "open database failed").WithTag(aerr.InternalError)
	}

	return nil
}

func (r *Database) RegisterMetrics(queryTime bool) {
	db := r.dbimpl.GetDB()
	if db == nil {
		panic("db not connected")
	}

	// gather stats from database
	prometheus.DefaultRegisterer.MustRegister(collectors.NewDBStatsCollector(db, "main"))

	if queryTime {
		r.queryDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Tracks the latencies for database query.",
				Buckets: []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"caller"},
		)

		prometheus.DefaultRegisterer.MustRegister(r.queryDuration)
	}
}

// Shutdown close database. Called by samber/do.
func (r *Database) Shutdown(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Info().Msg("closing db...")

	if err := r.dbimpl.Close(ctx); err != nil {
		return fmt.Errorf("close db error: %w", err)
	}

	logger.Debug().Msg("db closed")

	return nil
}

func (r *Database) Clear(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("migration start")

	err := r.dbimpl.Clear(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "migration error")
	}

	logger.Debug().Msg("migration finished")

	return nil
}

func (r *Database) Migrate(ctx context.Context) error {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("migration start")

	err := r.dbimpl.Migrate(ctx)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "migration error")
	}

	logger.Debug().Msg("migration finished")

	return nil
}

func (r *Database) getConnection(ctx context.Context) (*sqlx.Conn, error) {
	conn, err := r.dbimpl.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	return conn, nil
}

func (r *Database) closeConnection(ctx context.Context, conn *sqlx.Conn) {
	if err := r.dbimpl.CloseConnection(ctx, conn); err != nil {
		log.Logger.Error().Err(err).Msg("close connection failed")
	}
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

// InConnectionR run `fun` in database context. Open/close connection. Return `fun` result and error.
func InConnectionR[T any](ctx context.Context, r *Database, //nolint:ireturn
	fun func(context.Context) (T, error),
) (T, error) {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.getConnection(ctx)
	if err != nil {
		return *new(T), err
	}

	defer r.closeConnection(ctx, conn)

	ctx = WithCtx(ctx, conn)

	res, err := fun(ctx)
	if err != nil {
		return res, err
	}

	return res, nil
}

func InTransaction(ctx context.Context, r *Database, fun func(context.Context) error) error {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.getConnection(ctx)
	if err != nil {
		return err
	}

	defer r.closeConnection(ctx, conn)

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	ctx = WithCtx(ctx, tx)

	err = fun(ctx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			merr := errors.Join(err, fmt.Errorf("rollback error: %w", err))

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
func InTransactionR[T any](ctx context.Context, r *Database, //nolint:ireturn
	fun func(context.Context) (T, error),
) (T, error) {
	start := time.Now()
	defer r.observeQueryDuration(start)

	conn, err := r.getConnection(ctx)
	if err != nil {
		return *new(T), err
	}

	defer r.closeConnection(ctx, conn)

	tx, err := conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return *new(T), aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	ctx = WithCtx(ctx, tx)

	res, err := fun(ctx)
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

// ------------------------------------------------------
