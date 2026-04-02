package srvsupport

//
// httpsupport.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func SessionUser(store session.Store) string {
	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}

// Wrap add context and logger to handler.
func Wrap(handler func(ctx context.Context, w http.ResponseWriter, r *http.Request,
	logger *zerolog.Logger),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := hlog.FromRequest(r)
		handler(ctx, w, r, logger)
	}
}

// WrapNamed add context and logger to handler. `name` is put as `handler` in logger context.
func WrapNamed(
	handler func(ctx context.Context, w http.ResponseWriter, r *http.Request, logger *zerolog.Logger),
	name string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := hlog.FromRequest(r).
			With().Str("handler", name).
			Logger()

		ctx := logger.WithContext(r.Context())
		r = r.WithContext(ctx)

		handler(ctx, w, r, &logger)
	}
}
