package templates

//
// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"io"
	"time"

	"github.com/samber/do/v2"
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
