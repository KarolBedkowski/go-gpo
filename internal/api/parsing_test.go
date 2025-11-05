package api

//
// parsing_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/kabes/go-gpo/internal/assert"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		experr   bool
	}{
		{"", time.Time{}, true},
		{"2025-11-21T21:59:45", time.Date(2025, 11, 21, 21, 59, 45, 0, time.UTC), false},
		{"2025-11-21z23", time.Time{}, true},
		{"2025-02-20T21:59:45Z0", time.Time{}, true},
		{"2025-02-20T21:59:45+00:00", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20T21:59:45Z", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20T21:59:45+01:00", time.Date(2025, 2, 20, 20, 59, 45, 0, time.UTC), false},
		{"2025-02-20 21:59:45", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC), false},
		{"1762356879", time.Date(2025, 11, 5, 15, 34, 39, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			res, err := parseDate(tt.input)
			if tt.experr {
				assert.Err(t, err)
			} else {
				assert.NoErr(t, err)
				assert.Equal(t, res, tt.expected)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    any
		expected time.Time
		experr   bool
	}{
		{"", time.Time{}, true},
		{"2025-11-21T21:59:45", time.Date(2025, 11, 21, 21, 59, 45, 0, time.UTC), false},
		{"2025-11-21z23", time.Time{}, true},
		{"2025-02-20T21:59:45Z0", time.Time{}, true},
		{"2025-02-20T21:59:45+00:00", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20T21:59:45Z", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20T21:59:45+01:00", time.Date(2025, 2, 20, 20, 59, 45, 0, time.UTC), false},
		{"2025-02-20 21:59:45", time.Date(2025, 2, 20, 21, 59, 45, 0, time.UTC), false},
		{"2025-02-20", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC), false},
		{"1762356879", time.Date(2025, 11, 5, 15, 34, 39, 0, time.UTC), false},
		{int(1762356879), time.Date(2025, 11, 5, 15, 34, 39, 0, time.UTC), false},
		{int32(1762356879), time.Date(2025, 11, 5, 15, 34, 39, 0, time.UTC), false},
		{int64(1762356879), time.Date(2025, 11, 5, 15, 34, 39, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			res, err := parseTimestamp(tt.input)
			if tt.experr {
				assert.Err(t, err)
			} else {
				assert.NoErr(t, err)
				assert.Equal(t, res, tt.expected)
			}
		})
	}
}
