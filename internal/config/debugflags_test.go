package config

//
// debugflags_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"testing"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		input       string
		expected    []DebugFlag
		notexpected []DebugFlag
	}{
		{"", []DebugFlag{}, []DebugFlag{DebugMsgBody, DebugDo, DebugRouter, DebugDo}},
		{"xxx", []DebugFlag{}, []DebugFlag{DebugMsgBody, DebugDo, DebugRouter, DebugDo}},
		{"all", []DebugFlag{DebugMsgBody, DebugDo, DebugRouter, DebugDo}, []DebugFlag{}},
		{"all,do,go", []DebugFlag{DebugMsgBody, DebugDo, DebugRouter, DebugDo}, []DebugFlag{}},
		{"do,go", []DebugFlag{DebugDo, DebugDo}, []DebugFlag{DebugMsgBody, DebugRouter}},
		{"go,do,router", []DebugFlag{DebugDo, DebugDo, DebugRouter}, []DebugFlag{DebugMsgBody}},
		{"go,do,router,logbody", []DebugFlag{DebugDo, DebugDo, DebugRouter, DebugMsgBody}, []DebugFlag{}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			df := NewDebugFLags(tt.input)
			for _, e := range tt.expected {
				assert.True(t, df.HasFlag(e))
			}
			for _, e := range tt.notexpected {
				assert.True(t, !df.HasFlag(e))
			}
		})
	}
}
