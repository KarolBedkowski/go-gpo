package db

// dbcontext.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// ------------------------------------------------------------------------------

// Queryer define interface for object used to query database.
type Queryer interface {
	sqlx.QueryerContext
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

type Interface interface {
	// sqlx.ExtContext
	sqlx.QueryerContext
	sqlx.PreparerContext
	sqlx.ExecerContext

	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

// ------------------------------------------------------------------------------

//nolint:gochecknoglobals
var ctxDBInterfaceKey = any("CtxDBAccessKey")

// WithCtx create new context with database access object.
func WithCtx(ctx context.Context, dbctx Interface) context.Context {
	db, ok := ctx.Value(ctxDBInterfaceKey).(Interface)
	if ok && db != nil {
		return ctx
	}

	return context.WithValue(ctx, ctxDBInterfaceKey, dbctx)
}

// Ctx return database access object from context.
func Ctx(ctx context.Context) (Interface, bool) { //nolint:ireturn
	value, ok := ctx.Value(ctxDBInterfaceKey).(Interface)
	if !ok || value == nil {
		return nil, false
	}

	return value, true
}

// MustCtx return database access object from context. Panic when not exists.
func MustCtx(ctx context.Context) Interface { //nolint:ireturn
	value, ok := ctx.Value(ctxDBInterfaceKey).(Interface)
	if !ok || value == nil {
		panic("no dbcontext in context")
	}

	return value
}
