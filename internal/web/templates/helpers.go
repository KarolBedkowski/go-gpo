package templates

//
// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code.forgejo.org/go-chi/session"
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

// ------------------------------------------------------------------------------

type PageContext struct {
	Webroot string
	Session session.Store
}

func (p *PageContext) Flash() FlashStore {
	if p.Session != nil {
		if flash, ok := p.Session.Get("_flash").(FlashStore); ok {
			_ = p.Session.Delete("_flash")

			return flash
		}
	}

	return nil
}

//------------------------------------------------------------------------------

type Renderer struct {
	webroot string
}

func NewRenderer(i do.Injector) (*Renderer, error) {
	return &Renderer{
		webroot: do.MustInvokeNamed[string](i, "server.webroot"),
	}, nil
}

func (r *Renderer) WritePage(w io.Writer, req *http.Request, p Page) {
	pctx := PageContext{
		Webroot: r.webroot,
		Session: session.GetSession(req),
	}
	WritePageTemplate(w, p, &pctx)
}

type SimplePage interface {
	Body(pctx *PageContext) string
}

func (r *Renderer) WriteSimplePage(w io.Writer, req *http.Request, p SimplePage) {
	pctx := PageContext{
		Webroot: r.webroot,
		Session: session.GetSession(req),
	}
	_, _ = w.Write([]byte(p.Body(&pctx)))
}

func (r *Renderer) WriteSimpleError(writer io.Writer, req *http.Request, status int, details string) {
	page := ErrorPage{
		Status:  status,
		Message: http.StatusText(status),
		Details: details,
		ReqID:   "",
	}

	if status == http.StatusInternalServerError {
		ctx := req.Context()

		if reqid, ok := hlog.IDFromCtx(ctx); ok {
			page.ReqID = reqid.String()
		}
	}

	pctx := PageContext{
		Webroot: r.webroot,
		Session: session.GetSession(req),
	}
	WritePageTemplate(writer, &page, &pctx)
}

func (r *Renderer) WriteNotFoundError(w io.Writer, req *http.Request, details string) {
	r.WriteSimpleError(w, req, http.StatusNotFound, details)
}

func (r *Renderer) WriteBadRequestError(w io.Writer, req *http.Request, details string) {
	r.WriteSimpleError(w, req, http.StatusBadRequest, details)
}

func (r *Renderer) WriteError(w io.Writer, req *http.Request, err error) {
	switch {
	case aerr.HasTag(err, aerr.InternalError):
		r.WriteSimpleError(w, req, http.StatusInternalServerError, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.NotFound):
		r.WriteSimpleError(w, req, http.StatusNotFound, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.ValidationError) || aerr.HasTag(err, aerr.BadRequest):
		r.WriteSimpleError(w, req, http.StatusBadRequest, aerr.GetUserMessage(err))
	default:
		r.WriteSimpleError(w, req, http.StatusInternalServerError, aerr.GetUserMessage(err))
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

//------------------------------------------------------------------------------

type Flash struct {
	Level    string
	Messages []string
}

func (f *Flash) String() string {
	return fmt.Sprintf("Flash %s: %v", f.Level, f.Messages)
}

type FlashStore []*Flash

func AddFlash(req *http.Request, level, message string) {
	sess := session.GetSession(req)

	flashstore, ok := sess.Get("_flash").(FlashStore)
	if ok && flashstore != nil {
		for _, f := range flashstore {
			if f.Level == level {
				f.Messages = append(f.Messages, message)
				_ = sess.Set("_flash", flashstore)

				return
			}
		}
	}

	_ = sess.Set("_flash", append(flashstore, &Flash{level, []string{message}}))
}

func AddFlashError(w io.Writer, req *http.Request, err error) {
	switch {
	case aerr.HasTag(err, aerr.InternalError):
		msg := http.StatusText(http.StatusInternalServerError)
		if herr := aerr.GetUserMessage(err); herr != "" {
			msg += ": " + herr
		}

		if reqid, ok := hlog.IDFromCtx(req.Context()); ok {
			msg += " (reqid: " + reqid.String() + ")"
		}

		AddFlash(req, "error", msg)
	case aerr.HasTag(err, aerr.NotFound):
		msg := "Not found"
		if herr := aerr.GetUserMessage(err); herr != "" {
			msg = herr
		}

		AddFlash(req, "warn", msg)
	case aerr.HasTag(err, aerr.ValidationError):
		msg := "Validation error"
		if herr := aerr.GetUserMessage(err); herr != "" {
			msg += ": " + herr
		}

		AddFlash(req, "warn", msg)

	case aerr.HasTag(err, aerr.BadRequest):
		msg := http.StatusText(http.StatusBadRequest)
		if herr := aerr.GetUserMessage(err); herr != "" {
			msg += ": " + herr
		}

		AddFlash(req, "warn", msg)
	default:
		msg := http.StatusText(http.StatusInternalServerError)
		if herr := aerr.GetUserMessage(err); herr != "" {
			msg += ": " + herr
		}

		AddFlash(req, "error", msg)
	}
}
