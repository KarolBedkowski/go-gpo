package api

//
// parsing_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"bytes"
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

func TestParseOPML(t *testing.T) {
	data := bytes.NewBufferString(
		`<?xml version='1.0' encoding='UTF-8' standalone='no' ?>
<opml version="2.0">
  <head>
    <title>AntennaPod Subscriptions</title>
    <dateCreated>19 Oct 25 17:27:06 +0200</dateCreated>
  </head>
  <body>
    <outline text="yyy" title="title yyy" type="rss" xmlUrl="http://www.example.com/podcast1/podcast.xml" htmlUrl="http://podcast1.example.com" />
    <outline text="xxx" title="title xxx" type="rss" xmlUrl="http://www.example.com/podcast2/podcast.xml" htmlUrl="http://podcast2.example.com" />
    <outline text="zzz" title="title zzz" type="rss" xmlUrl="http://www.example.com/podcast3/podcast.xml" htmlUrl="http://podcast3.example.com" />
  </body>
</opml>`)

	urls, err := parseOPML(data)
	assert.NoErr(t, err)
	assert.EqualSorted(t, urls, []string{
		"http://www.example.com/podcast1/podcast.xml",
		"http://www.example.com/podcast2/podcast.xml",
		"http://www.example.com/podcast3/podcast.xml",
	})
}

func TestParseTextSubs(t *testing.T) {
	data := bytes.NewBufferString(`http://www.example.com/podcast1/podcast.xml
http://www.example.com/podcast2/podcast.xml
http://www.example.com/podcast3/podcast.xml`)
	urls, err := parseTextSubs(data)
	assert.NoErr(t, err)
	assert.EqualSorted(t, urls, []string{
		"http://www.example.com/podcast1/podcast.xml",
		"http://www.example.com/podcast2/podcast.xml",
		"http://www.example.com/podcast3/podcast.xml",
	})
}
