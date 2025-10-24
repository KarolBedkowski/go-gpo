//
// parsing.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package api

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/oxtyped/go-opml/opml"
)

// ---------------------------------------

type ParseError struct {
	msg string
}

func NewParseError(msg string, args ...any) ParseError {
	return ParseError{fmt.Sprintf(msg, args...)}
}

func (p ParseError) Error() string {
	return p.msg
}

// ---------------------------------------

var dateFormats = []string{
	"2006-01-02T15:04:05",
	time.RFC3339,
	time.RFC3339Nano,
	time.DateTime,
	time.DateOnly,
}

func parseDate(str string) (time.Time, error) {
	for _, df := range dateFormats {
		ts, err := time.Parse(df, str)
		if err == nil {
			return ts, nil
		}
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return time.Unix(val, 0), nil
	}

	return time.Time{}, NewParseError("cant parse %q as date", str)
}

func parseTimestamp(timestamp any) (time.Time, error) {
	switch v := timestamp.(type) {
	case int:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case int32:
		return time.Unix(int64(v), 0), nil
	case string:
		if ts, err := parseDate(v); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, NewParseError("cant parse timestamp %v", timestamp)
}

func parseOPML(r io.Reader) ([]string, error) {
	// TODO need tests
	var buf bytes.Buffer

	count, err := io.Copy(&buf, r)
	if err != nil {
		return nil, fmt.Errorf("parse opml read error: %w", err)
	}

	if count == 0 {
		return []string{}, nil
	}

	o, err := opml.NewOPML(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse opml error: %w", err)
	}

	subs := make([]string, 0)

	for _, i := range o.Outlines() {
		if url := i.XMLURL; url != "" {
			subs = append(subs, url)
		}
	}

	return subs, nil
}
