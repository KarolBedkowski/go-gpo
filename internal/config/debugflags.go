package config

import (
	"slices"
	"strings"
)

//
// debugflags.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

//-------------------------------------------------------------

type DebugFlag string

const (
	DebugMsgBody = DebugFlag("logbody")
	DebugDo      = DebugFlag("do")
	DebugGo      = DebugFlag("go")
)

type DebugFlags []string

func NewDebugFLags(flags string) DebugFlags {
	return DebugFlags(strings.Split(flags, ","))
}

func (d DebugFlags) HasFlag(flag DebugFlag) bool {
	return slices.Contains(d, "all") || slices.Contains(d, string(flag))
}
