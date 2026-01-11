//go:build trace

package common

//
// tracing.go
// Copyright (C) 2026 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"runtime/trace"
	"strings"

	xtrace "golang.org/x/net/trace"
)

const TracingAvailable = true

//-------------------------------------------------------------

func TraceLazyPrintf(ctx context.Context, format string, a ...any) {
	if trace.IsEnabled() {
		if cat, _, ok := strings.Cut(format, ":"); ok {
			trace.Logf(ctx, cat, format, a...)
		} else {
			trace.Logf(ctx, "", format, a...)
		}
	}

	if tr, ok := xtrace.FromContext(ctx); ok && tr != nil {
		tr.LazyPrintf(format, a...)
	}
}

func TraceErrorLazyPrintf(ctx context.Context, format string, a ...any) {
	if trace.IsEnabled() {
		if cat, _, ok := strings.Cut(format, ":"); ok {
			trace.Logf(ctx, "error "+cat, format, a...)
		} else {
			trace.Logf(ctx, "error", format, a...)
		}
	}

	if tr, ok := xtrace.FromContext(ctx); ok && tr != nil {
		tr.LazyPrintf(format, a...)
		tr.SetError()
	}
}

//-------------------------------------------------------------

type EventLog struct {
	events xtrace.EventLog
}

func NewEventLog(pkg, domain string) *EventLog {
	return &EventLog{xtrace.NewEventLog(pkg, domain)}
}

func (e *EventLog) Printf(format string, a ...any) {
	if e != nil && e.events != nil {
		e.events.Printf(format, a...)
	}
}

func (e *EventLog) Errorf(format string, a ...any) {
	if e != nil && e.events != nil {
		e.events.Errorf(format, a...)
	}
}

func (f *EventLog) Close() {
	f.events.Finish()
}

//-------------------------------------------------------------

//nolint:gochecknoglobals
var ctxEventLogKey = any("ctxEventLogKey")

// ContextEventLog return device name from context.
func ContextEventLog(ctx context.Context) *EventLog {
	value, ok := ctx.Value(ctxEventLogKey).(*EventLog)
	if ok {
		return value
	}

	return nil
}

// ContextWithEventLog create context with device name.
func ContextWithEventLog(ctx context.Context, eventlog *EventLog) context.Context {
	return context.WithValue(ctx, ctxEventLogKey, eventlog)
}

// ------------------------------------------------------

func NewCtxEventLog(ctx context.Context, pkg, domain string) (context.Context, func()) {
	e := NewEventLog(pkg, domain)

	return ContextWithEventLog(ctx, e), e.Close
}

func EventLogPrintf(ctx context.Context, format string, a ...any) {
	if e, ok := ctx.Value(ctxEventLogKey).(*EventLog); ok && e != nil && e.events != nil {
		e.events.Printf(format, a...)
	}
}

func EventLogErrorff(ctx context.Context, format string, a ...any) {
	if e, ok := ctx.Value(ctxEventLogKey).(*EventLog); ok && e != nil && e.events != nil {
		e.events.Errorf(format, a...)
	}
}

//-------------------------------------------------------------

type Region struct {
	r *trace.Region
}

func NewRegion(ctx context.Context, regionType string) Region {
	return Region{trace.StartRegion(ctx, regionType)}
}

func (r Region) End() {
	r.r.End()
}

func NewTask(ctx context.Context, taskType string) (context.Context, func()) {
	ctx, task := trace.NewTask(ctx, taskType)

	return ctx, task.End
}
