package srvsupport

//
// httpsupport.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.forgejo.org/go-chi/session"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
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

// WriteError decode and write error to ResponseWriter.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case aerr.HasTag(err, aerr.InternalError):
		WriteSimpleError(w, r, http.StatusInternalServerError, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.NotFound):
		WriteSimpleError(w, r, http.StatusNotFound, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.ValidationError) || aerr.HasTag(err, aerr.BadRequest):
		WriteSimpleError(w, r, http.StatusBadRequest, aerr.GetUserMessage(err))
	default:
		WriteSimpleError(w, r, http.StatusInternalServerError, aerr.GetUserMessage(err))
	}
}

type errorData struct {
	Error   string `json:"error"                xml:"error"`
	ReqID   string `json:"request_id,omitempty" xml:"request_id,omitempty"`
	TS      string `json:"ts,omitempty"         xml:"ts,omitempty"`
	Details string `json:"details,omitempty"    xml:"details,omitempty"`

	status int `json:"-" xml:"-"`
}

func newErrorData(status int, details string) *errorData {
	return &errorData{Error: http.StatusText(status), Details: details, status: status}
}

func (e *errorData) withRequest(r *http.Request) {
	if rid, ok := hlog.IDFromRequest(r); ok {
		e.ReqID = rid.String()
	}
}

func (e *errorData) writeJSON(w http.ResponseWriter, r *http.Request) {
	render.Status(r, e.status)
	render.JSON(w, r, e)
}

func (e *errorData) writePlain(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, e.Error, e.status)

	if e.Details != "" {
		fmt.Fprintln(w, e.Details)
	}

	if e.ReqID != "" {
		fmt.Fprintln(w, "reqid="+e.ReqID)
	}

	if e.TS != "" {
		fmt.Fprintln(w, "ts="+e.TS)
	}
}

func (e *errorData) writeXML(w http.ResponseWriter, r *http.Request) {
	render.Status(r, e.status)
	render.XML(w, r, e)
}

func (e *errorData) withTS() {
	e.TS = time.Now().Format(time.RFC3339Nano)
}

func WriteSimpleError(w http.ResponseWriter, r *http.Request, status int, details string) {
	edata := newErrorData(status, details)

	if status == http.StatusInternalServerError {
		edata.withRequest(r)
		edata.withTS()
	}

	switch reqContentType := r.Header.Get("Content-Type"); {
	case strings.HasPrefix(reqContentType, "application/json"):
		edata.writeJSON(w, r)
	case strings.HasPrefix(reqContentType, "text/xml") || strings.Contains(reqContentType, "opml"):
		edata.writeXML(w, r)
	default:
		edata.writePlain(w, r)
	}
}
