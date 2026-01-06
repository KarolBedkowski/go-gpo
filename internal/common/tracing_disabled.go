//go:build !trace

package common

//
// tracing_disabled.go
// Copyright (C) 2026 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"

	"golang.org/x/net/trace"
)

const TracingAvailable = false

func WithTrace(ctx context.Context, callback func(trace.Trace)) {
}

func TraceLazyPrintf(ctx context.Context, format string, a ...any) {
}

func TraceErrorLazyPrintf(ctx context.Context, format string, a ...any) {
}

type EventLog struct{}

func NewEventLog(pkg, domain string) *EventLog {
	return &EventLog{}
}

func (e *EventLog) Printf(format string, a ...any) {
}

func (e *EventLog) Errorf(format string, a ...any) {
}

func (f *EventLog) Close() {
}
