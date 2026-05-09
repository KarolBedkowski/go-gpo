package web

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
	nt "gitlab.com/kabes/go-gpo/internal/web/templates"
)

type errorPages struct {
	renderer *nt.Renderer
}

func newErrorPages(i do.Injector) (errorPages, error) {
	return errorPages{
		renderer: do.MustInvoke[*nt.Renderer](i),
	}, nil
}

func (e errorPages) Register(router *chi.Mux) {
	router.NotFound(srvsupport.WrapNamed(e.notfoundHandler, "web_notfound"))
	router.MethodNotAllowed(srvsupport.WrapNamed(e.methodNotAllowedHandler, "web_methodnotallowed"))
}

func (e errorPages) notfoundHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	sess := session.GetSession(r)
	user := srvsupport.SessionUser(sess)

	if user == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	} else {
		e.renderer.WriteNotFoundError(ctx, w, "")
	}
}

func (e errorPages) methodNotAllowedHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	logger *zerolog.Logger,
) {
	sess := session.GetSession(r)
	user := srvsupport.SessionUser(sess)

	if user == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	} else {
		e.renderer.WriteBadRequestError(ctx, w, "")
	}
}
