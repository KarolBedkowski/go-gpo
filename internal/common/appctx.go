package common

import (
	"context"

	"golang.org/x/net/trace"
)

//nolint:gochecknoglobals
var ctxUserKey = any("ctxUserKey")

// ContextUser return user name from context.
func ContextUser(ctx context.Context) string {
	suser, ok := ctx.Value(ctxUserKey).(string)
	if ok {
		return suser
	}

	return ""
}

// ContextWithUser create new context with user name.
func ContextWithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, ctxUserKey, username)
}

// ------------------------------------------------------

//nolint:gochecknoglobals
var ctxDeviceKey = any("ctxDeviceKey")

// ContextDevice return device name from context.
func ContextDevice(ctx context.Context) string {
	value, ok := ctx.Value(ctxDeviceKey).(string)
	if ok {
		return value
	}

	return ""
}

// ContextWithDevice create context with device name.
func ContextWithDevice(ctx context.Context, devicename string) context.Context {
	return context.WithValue(ctx, ctxDeviceKey, devicename)
}

// ------------------------------------------------------

func WithTrace(ctx context.Context, callback func(trace.Trace)) {
	if tr, ok := trace.FromContext(ctx); ok {
		callback(tr)
	}
}

func TraceLazyPrintf(ctx context.Context, format string, a ...any) {
	if tr, ok := trace.FromContext(ctx); ok && tr != nil {
		tr.LazyPrintf(format, a...)
	}
}
