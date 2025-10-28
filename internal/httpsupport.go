package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitea.com/go-chi/session"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

//
// httpsupport.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

// GetSinceParameter from request url query.
func GetSinceParameter(r *http.Request) (time.Time, error) {
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

func SessionUser(store session.Store) string {
	log.Debug().Str("session_id", store.ID()).Msg("session id")

	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}

// internal.Wrap add context and logger to handler.
func Wrap(handler func(ctx context.Context, w http.ResponseWriter, r *http.Request,
	logger *zerolog.Logger),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := hlog.FromRequest(r)
		handler(ctx, w, r, logger)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, code int, err error) {
	var msg string
	if err == nil {
		msg = http.StatusText(code)
	} else {
		msg = err.Error()
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		res := struct {
			Error string `json:"error"`
		}{msg}

		render.Status(r, code)
		render.JSON(w, r, &res)

		return
	}

	http.Error(w, msg, code)
}
