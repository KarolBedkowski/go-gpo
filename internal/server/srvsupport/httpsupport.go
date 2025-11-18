package srvsupport

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"gitea.com/go-chi/session"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//
// httpsupport.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

func SessionUser(store session.Store) string {
	log.Debug().Str("session_id", store.ID()).Msg("session id")

	suserint := store.Get("user")
	if username, ok := suserint.(string); ok {
		return username
	}

	return ""
}

// srvsupport.Wrap add context and logger to handler.
func Wrap(handler func(ctx context.Context, w http.ResponseWriter, r *http.Request,
	logger *zerolog.Logger),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := hlog.FromRequest(r)
		handler(ctx, w, r, logger)
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
		render.JSON(w, r, &res)

		return
	}

	http.Error(w, msg, code)
}

// CheckAndWriteError decode and write error to ResponseWriter.
func CheckAndWriteError(w http.ResponseWriter, r *http.Request, err error) {
	msg := aerr.GetUserMessage(err)

	switch {
	case errors.Is(err, service.ErrUnknownDevice):
		WriteError(w, r, http.StatusNotFound, msg)

	case aerr.HasTag(err, aerr.InternalError):
		// write message if is defined in error
		WriteError(w, r, http.StatusInternalServerError, msg)

	case aerr.HasTag(err, aerr.DataError):
		WriteError(w, r, http.StatusBadRequest, msg)

	default:
		// unknown error; newer show details
		WriteError(w, r, http.StatusInternalServerError, "")
	}
}
