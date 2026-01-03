package srvsupport

//
// httpsupport.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"gitea.com/go-chi/session"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
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

func WriteError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if msg == "" {
		msg = http.StatusText(code)
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		res := struct {
			Error string `json:"error"`
		}{msg}

		render.Status(r, code)
		RenderJSON(w, r, &res)

		return
	}

	http.Error(w, msg, code)
}

// CheckAndWriteError decode and write error to ResponseWriter.
func CheckAndWriteError(w http.ResponseWriter, r *http.Request, err error) {
	msg := aerr.GetUserMessage(err)

	switch {
	case errors.Is(err, common.ErrUnknownDevice):
		WriteError(w, r, http.StatusNotFound, msg)

	case aerr.HasTag(err, aerr.InternalError):
		// write message if is defined in error
		WriteError(w, r, http.StatusInternalServerError, msg)

	case aerr.HasTag(err, aerr.ValidationError):
		WriteError(w, r, http.StatusBadRequest, msg)

	case aerr.HasTag(err, aerr.DataError):
		WriteError(w, r, http.StatusBadRequest, msg)

	default:
		// unknown error; newer show details
		WriteError(w, r, http.StatusInternalServerError, "")
	}
}
