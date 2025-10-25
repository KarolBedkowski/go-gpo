package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

//
// parsing.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

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

	return time.Time{}, fmt.Errorf("cant parse %q as date", str)
}

func sinceFromParameter(r *http.Request) (time.Time, error) {
	since := time.Time{}
	if s := r.URL.Query().Get("since"); s != "" {
		se, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return since, fmt.Errorf("parse since %q error: %w", s, err)
		}

		since = time.Unix(se, 0)
	}

	return since, nil
}
