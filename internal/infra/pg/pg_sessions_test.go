package pg

//
// sqlite_sessions_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"testing"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestSessionsEncode(t *testing.T) {
	tc := []map[any]any{
		{},
		{"aa": 123, 123: 456},
		{"123": []int16{1, 2, 3}, "abc": map[int]string{1: "a", 2: "b"}},
		{"123": "abc", "abc": map[string]any{"a": 12, "b": 321}},
	}

	for i, inp := range tc {
		t.Run(fmt.Sprintf("TestSessionsEncode-%d", i), func(t *testing.T) {
			enc, err := encodeSession(inp)
			assert.NoErr(t, err)

			dec, err := decodeSession(enc)
			assert.NoErr(t, err)

			assert.Equal(t, dec, inp)
		})
	}
}
