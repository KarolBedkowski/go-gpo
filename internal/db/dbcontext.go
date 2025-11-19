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

var CtxDBInterfaceKey = any("CtxDBAccessKey")

func WithCtx(ctx context.Context, dbctx Interface) context.Context {
	db, ok := ctx.Value(CtxDBInterfaceKey).(Interface)
	if ok && db != nil {
		return ctx
	}

	return context.WithValue(ctx, CtxDBInterfaceKey, dbctx)
}

func Ctx(ctx context.Context) (Interface, bool) {
	value, ok := ctx.Value(CtxDBInterfaceKey).(Interface)
	if !ok || value == nil {
		return nil, false
	}

	return value, true
}

func MustCtx(ctx context.Context) Interface {
	value, ok := ctx.Value(CtxDBInterfaceKey).(Interface)
	if !ok || value == nil {
		panic("no dbcontext in context")
	}

	return value
}
