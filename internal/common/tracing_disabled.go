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
)

const TracingAvailable = false

//-------------------------------------------------------------

func TraceLazyPrintf(ctx context.Context, format string, a ...any)      {}
func TraceErrorLazyPrintf(ctx context.Context, format string, a ...any) {}

//-------------------------------------------------------------

type EventLog struct{}

func NewEventLog(pkg, domain string) *EventLog {
	return &EventLog{}
}

func (e *EventLog) Printf(format string, a ...any) {}
func (e *EventLog) Errorf(format string, a ...any) {}
func (f *EventLog) Close()                         {}

//-------------------------------------------------------------

// ContextEventLog return device name from context.
func ContextEventLog(ctx context.Context) *EventLog {
	return nil
}

// ContextWithEventLog create context with device name.
func ContextWithEventLog(ctx context.Context, eventlog *EventLog) context.Context {
	return ctx
}

// ------------------------------------------------------

func NewCtxEventLog(ctx context.Context, pkg, domain string) (context.Context, func()) {
	return ctx, func() {}
}

func EventLogPrintf(ctx context.Context, format string, a ...any)  {}
func EventLogErrorff(ctx context.Context, format string, a ...any) {}

// -------------------------------------------------------------
type Region struct{}

func NewRegion(ctx context.Context, regionType string) Region {
	return Region{}
}

func (r Region) End() {}

func NewTask(ctx context.Context, taskType string) (context.Context, func()) {
	return ctx, func() {}
}
