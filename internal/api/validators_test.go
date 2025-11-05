package api

//
// validators_test.go
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
		input    string
		expected string
	}{
		{"", ""},
		{"       ", ""},
		{" http://abc.com/abc ", "http://abc.com/abc"},
		{"http://xxx.yyy/xxx?123&dkdkd ", "http://xxx.yyy/xxx?123&dkdkd"},
		{"https://xxx.yyy/xxx?123&dkdkd ", "https://xxx.yyy/xxx?123&dkdkd"},
		{"ftp://xxx.yyy/xxx?123&dkdkd ", ""},
		{"xxx.yyy/xxx?123&dkdkd ", ""},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			res := SanitizeURL(tt.input)
			assert.Equal(t, res, tt.expected)
		})
	}
}

func TestSanitizeURLs(t *testing.T) {
	tests := []struct {
		input           []string
		expected        []string
		expectedchanges [][]string
	}{
		{
			[]string{" http://abc.com/abc "},
			[]string{"http://abc.com/abc"},
			[][]string{{" http://abc.com/abc ", "http://abc.com/abc"}},
		},
		{
			[]string{"http://abc.com/abc", "ddsldsk"},
			[]string{"http://abc.com/abc"},
			[][]string{},
		},
		{
			[]string{" ", "http://abc.com/abc", "ddsldsk", "ftp://123.233.3"},
			[]string{"http://abc.com/abc"},
			[][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			res, changes := SanitizeURLs(tt.input)
			assert.Equal(t, res, tt.expected)
			assert.Equal(t, changes, tt.expectedchanges)
		})
	}
}
