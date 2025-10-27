// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

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

// ensureList create empty list if `inp` is null or return `inp` otherwise.
func ensureList[T any](inp []T) []T {
	if inp == nil {
		return make([]T, 0)
	}

	return inp
}

// wrap add context and logger to handler.
func wrap(handler func(ctx context.Context, w http.ResponseWriter, r *http.Request,
	logger *zerolog.Logger),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := hlog.FromRequest(r)
		handler(ctx, w, r, logger)
	}
}

func sessionUser(store session.Store) string {
	log.Debug().Str("session_id", store.ID()).Msg("session id")

	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}
