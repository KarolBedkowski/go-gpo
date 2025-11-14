// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ensureList create empty list if `inp` is null or return `inp` otherwise.
// func ensureList[T any](inp []T) []T {
// 	if inp == nil {
// 		return make([]T, 0)
// 	}

// 	return inp
// }

// getSinceParameter from request url query.
func getSinceParameter(r *http.Request) (time.Time, error) {
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
