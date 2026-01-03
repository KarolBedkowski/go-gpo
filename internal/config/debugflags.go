package config

//
// debugflags.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/common"
)

//-------------------------------------------------------------

type DebugFlag string

const (
	// DebugMsgBody enable logging request and response body and headers.
	DebugMsgBody = DebugFlag("logbody")
	// DebugDo enable logging samber/do and /debug/do endpoint.
	DebugDo = DebugFlag("do")
	// DebugGo enable /debug/pprof endpoint.
	DebugGo = DebugFlag("go")
	// DebugRouter show defined routes.
	DebugRouter = DebugFlag("router")
	// DebugDBQueryMetrics enable metrics for query metrics.
	DebugDBQueryMetrics = DebugFlag("querymetrics")
	// DebugFlightRecorder enable flight recorder for long server queries.
	DebugFlightRecorder = DebugFlag("flighrecorder")
	// DebugTrace enable tracing with net/trace.
	DebugTrace = DebugFlag("trace")

	// DebugAll enable all debug flags.
	DebugAll = DebugFlag("all")
	// DebugNone disable all debug flags.
	DebugNone = DebugFlag("")
)

type DebugFlags []string

func NewDebugFLags(flags string) DebugFlags {
	df := DebugFlags(strings.Split(flags, ","))

	if !common.TracingAvailable && (df.HasFlag(DebugTrace) || df.HasFlag(DebugFlightRecorder)) {
		log.Logger.Warn().Msg("FlightRecorder and tracing disabled due to compilation tag")
	}

	return df
}

func (d DebugFlags) HasFlag(flag DebugFlag) bool {
	return slices.Contains(d, "all") || slices.Contains(d, string(flag))
}
