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

func WithTrace(ctx context.Context, callback func(xtrace.Trace)) {
	if tr, ok := xtrace.FromContext(ctx); ok {
		callback(tr)
	}
}

func TraceLazyPrintf(ctx context.Context, format string, a ...any) {
	if trace.IsEnabled() {
		cat, rest, ok := strings.Cut(format, ":")
		if ok {
			rest = strings.TrimSpace(rest)
			trace.Logf(ctx, cat, rest, a...)
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
		cat, rest, ok := strings.Cut(format, ":")
		if ok {
			rest = strings.TrimSpace(rest)
			trace.Logf(ctx, "error "+cat, rest, a...)
		} else {
			trace.Logf(ctx, "error", format, a...)
		}
	}

	if tr, ok := xtrace.FromContext(ctx); ok && tr != nil {
		tr.LazyPrintf(format, a...)
		tr.SetError()
	}
}

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

type Region struct {
	r *trace.Region
}

func NewRegion(ctx context.Context, regionType string) Region {
	return Region{trace.StartRegion(ctx, regionType)}
}

func (r Region) End() {
	r.r.End()
}
