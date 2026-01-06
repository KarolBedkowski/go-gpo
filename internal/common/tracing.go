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

	"golang.org/x/net/trace"
)

const TracingAvailable = true

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

func TraceErrorLazyPrintf(ctx context.Context, format string, a ...any) {
	if tr, ok := trace.FromContext(ctx); ok && tr != nil {
		tr.LazyPrintf(format, a...)
		tr.SetError()
	}
}

type EventLog struct {
	events trace.EventLog
}

func NewEventLog(pkg, domain string) *EventLog {
	return &EventLog{trace.NewEventLog(pkg, domain)}
}

func (e *EventLog) Printf(format string, a ...any) {
	e.events.Printf(format, a...)
}

func (e *EventLog) Errorf(format string, a ...any) {
	e.events.Errorf(format, a...)
}

func (f *EventLog) Close() {
	f.events.Finish()
}
