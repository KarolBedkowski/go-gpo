package templates

//
// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/hlog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

func formatDateTime(t time.Time) string {
	return t.Format(time.DateTime)
}

func formatPInt32AsDuration(v *int32) string {
	if v == nil {
		return ""
	}

	return (time.Duration(int(*v)) * time.Second).String()
}

type PageContext struct {
	Webroot string
}

type Renderer struct {
	pageContext *PageContext
}

func NewRenderer(i do.Injector) (*Renderer, error) {
	return &Renderer{
		&PageContext{
			Webroot: do.MustInvokeNamed[string](i, "server.webroot"),
		},
	}, nil
}

func (r *Renderer) WritePage(w io.Writer, p Page) {
	WritePageTemplate(w, p, r.pageContext)
}

type SimplePage interface {
	Body(pctx *PageContext) string
}

func (r *Renderer) WriteSimplePage(w io.Writer, p SimplePage) {
	_, _ = w.Write([]byte(p.Body(r.pageContext)))
}

//------------------------------------------------------------------------------

func (r *Renderer) WriteSimpleError(ctx context.Context, writer io.Writer, status int, details string) {
	page := ErrorPage{
		Status:  status,
		Message: http.StatusText(status),
		Details: details,
		ReqID:   "",
	}

	if status == http.StatusInternalServerError {
		if reqid, ok := hlog.IDFromCtx(ctx); ok {
			page.ReqID = reqid.String()
		}
	}

	WritePageTemplate(writer, &page, r.pageContext)
}

func (r *Renderer) WriteNotFoundError(ctx context.Context, w io.Writer, details string) {
	r.WriteSimpleError(ctx, w, http.StatusNotFound, details)
}

func (r *Renderer) WriteBadRequestError(ctx context.Context, w io.Writer, details string) {
	r.WriteSimpleError(ctx, w, http.StatusBadRequest, details)
}

func (r *Renderer) WriteError(ctx context.Context, w io.Writer, err error) {
	switch {
	case aerr.HasTag(err, aerr.InternalError):
		r.WriteSimpleError(ctx, w, http.StatusInternalServerError, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.NotFound):
		r.WriteSimpleError(ctx, w, http.StatusNotFound, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.ValidationError) || aerr.HasTag(err, aerr.BadRequest):
		r.WriteSimpleError(ctx, w, http.StatusBadRequest, aerr.GetUserMessage(err))
	default:
		r.WriteSimpleError(ctx, w, http.StatusInternalServerError, aerr.GetUserMessage(err))
	}
}

//------------------------------------------------------------------------------

func shortString(str string, maxlen int) string {
	if len(str) <= maxlen {
		return str
	}

	str = str[:maxlen]

	if lastSep := strings.LastIndexAny(str, " \t\n\r"); lastSep > -1 {
		str = str[:lastSep]
	}

	return str + "…"
}
