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
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

//------------------------------------------------------------------------------

type queryObserver struct {
	queryDuration *prometheus.HistogramVec
}

func (q *queryObserver) observeQueryDuration(start time.Time) {
	if q.queryDuration == nil {
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
	q.queryDuration.WithLabelValues(caller).Observe(time.Since(start).Seconds())
}

var observer = queryObserver{} //nolint: gochecknoglobals

func RegisterMetrics(i do.Injector, queryTime bool) {
	db := do.MustInvoke[*sql.DB](i)

	// gather stats from database
	prometheus.DefaultRegisterer.MustRegister(collectors.NewDBStatsCollector(db, "main"))

	if queryTime {
		observer.queryDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Tracks the latencies for database query.",
				Buckets: []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"caller"},
		)

		prometheus.DefaultRegisterer.MustRegister(observer.queryDuration)
	}
}

//------------------------------------------------------------------------------

// InConnectionR run `fun` in database context. Open/close connection. Return `fun` result and error.
func InConnectionR[T any](ctx context.Context, database repository.Database, //nolint:ireturn
	fun func(context.Context) (T, error),
) (T, error) {
	logger := log.Ctx(ctx)

	start := time.Now()
	defer observer.observeQueryDuration(start)

	defer common.NewRegion(ctx, "InConnectionR").End()

	conn, err := database.GetConnection(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: get connection error=%q", err)

		return *new(T), aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	defer func() {
		if err := database.CloseConnection(ctx, conn); err != nil {
			logger.Error().Err(err).Msgf("db.InConnectionR: close connection failed error=%q", err)
		}
	}()

	common.TraceLazyPrintf(ctx, "DB: got connection")

	ctx = WithCtx(ctx, conn)

	res, err := fun(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: exec func error=%q", err)

		return res, err
	}

	common.TraceLazyPrintf(ctx, "DB: finished")

	return res, nil
}

func InTransaction(ctx context.Context, database repository.Database, fun func(context.Context) error) error {
	logger := log.Ctx(ctx)

	start := time.Now()
	defer observer.observeQueryDuration(start)

	defer common.NewRegion(ctx, "InTransaction").End()

	conn, err := database.GetConnection(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: got connection error=%q", err)

		return aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	common.TraceLazyPrintf(ctx, "DB: got connection")

	defer func() {
		if err := database.CloseConnection(ctx, conn); err != nil {
			logger.Error().Err(err).Msgf("db.InTransaction: close connection failed error=%q", err)
		}
	}()

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: begin tx error=%q", err)

		return aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	common.TraceLazyPrintf(ctx, "DB: got tx")

	ctx = WithCtx(ctx, tx)

	err = fun(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: exec func error=%q", err)

		if err := tx.Rollback(); err != nil {
			return errors.Join(err, aerr.ApplyFor(aerr.ErrDatabase, err, "execute func and rollback error"))
		}

		return err
	}

	common.TraceLazyPrintf(ctx, "DB: exec func success")

	if err := tx.Commit(); err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: commit error=%q", err)

		return aerr.ApplyFor(aerr.ErrDatabase, err, "commit tx failed")
	}

	common.TraceLazyPrintf(ctx, "DB: committed")

	return nil
}

// InTransactionR run `fun` in db transactions; return `fun` result and error.
func InTransactionR[T any](ctx context.Context, database repository.Database, //nolint:ireturn
	fun func(context.Context) (T, error),
) (T, error) {
	logger := log.Ctx(ctx)

	start := time.Now()
	defer observer.observeQueryDuration(start)

	defer common.NewRegion(ctx, "InTransactionR").End()

	common.TraceLazyPrintf(ctx, "DB: start")

	conn, err := database.GetConnection(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: get connection error=%q", err)

		return *new(T), aerr.ApplyFor(aerr.ErrDatabase, err, "failed open connection")
	}

	common.TraceLazyPrintf(ctx, "DB: got connection")

	defer func() {
		if err := database.CloseConnection(ctx, conn); err != nil {
			logger.Error().Err(err).Msgf("db.InTransactionR: close connection failed error=%q", err)
		}
	}()

	tx, err := conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: begin tx error=%q", err)

		return *new(T), aerr.ApplyFor(aerr.ErrDatabase, err, "begin tx failed")
	}

	common.TraceLazyPrintf(ctx, "DB: got tx")

	ctx = WithCtx(ctx, tx)

	res, err := fun(ctx)
	if err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: exec func error=%q", err)

		if err := tx.Rollback(); err != nil {
			return res, errors.Join(err, aerr.ApplyFor(aerr.ErrDatabase, err, "execute func and rollback error"))
		}

		return res, err
	}

	common.TraceLazyPrintf(ctx, "DB: exec func success")

	if err := tx.Commit(); err != nil {
		common.TraceErrorLazyPrintf(ctx, "DB: commit error=%q", err)

		return res, aerr.ApplyFor(aerr.ErrDatabase, err, "commit tx failed")
	}

	common.TraceLazyPrintf(ctx, "DB: committed")

	return res, nil
}

//------------------------------------------------------------------------------
