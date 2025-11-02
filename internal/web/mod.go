package web

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type WEB struct {
	devicePages  devicePages
	userPages    userPages
	episodePages episodePages
	podcastPages podcastPages
	template     templates
	webroot      string
}

func New(i do.Injector) (WEB, error) {
	webroot := do.MustInvokeNamed[string](i, "server.webroot")

	return WEB{
		devicePages:  do.MustInvoke[devicePages](i),
		userPages:    do.MustInvoke[userPages](i),
		episodePages: do.MustInvoke[episodePages](i),
		podcastPages: do.MustInvoke[podcastPages](i),
		webroot:      webroot,
		template:     do.MustInvoke[templates](i),
	}, nil
}

func (w *WEB) Routes() chi.Router {
	router := chi.NewRouter()

	router.Get("/", internal.Wrap(w.indexPage))
	router.Mount("/device", w.devicePages.Routes())
	router.Mount("/podcast", w.podcastPages.Routes())
	router.Mount("/episode", w.episodePages.Routes())
	router.Mount("/user", w.userPages.Routes())

	fs := http.FileServerFS(staticFS)
	router.Method("GET", "/static/*", http.StripPrefix("/web/", fs))

	return router
}

func (w *WEB) indexPage(ctx context.Context, writer http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	_ = ctx

	if err := w.template.executeTemplate(writer, "index.tmpl", nil); err != nil {
		logger.Error().Err(err).Str("mod", "web").Msg("execute template error")
		internal.WriteError(writer, r, http.StatusInternalServerError, nil)
	}
}

//-----------------------------------------------

type templates map[string]*template.Template

// newTemplate loads templates.
func newTemplates(webroot string) templates {
	logger := log.Logger.With().Str("mod", "web").Logger()
	funcs := template.FuncMap{"webroot": func() string { return webroot }}
	base := template.Must(template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/_base*"))

	direntries, err := templatesFS.ReadDir("templates")
	if err != nil {
		panic(err)
	}

	res := make(map[string]*template.Template)

	for _, de := range direntries {
		if de.IsDir() {
			continue
		}

		name := de.Name()
		if name[0] == '_' {
			continue
		}

		logger.Debug().Msgf("loading template: %s", name)

		baseclone := template.Must(base.Clone())
		res[name] = template.Must(baseclone.ParseFS(templatesFS, "templates/"+name))
	}

	return res
}

func newTemplatesI(i do.Injector) (templates, error) {
	webroot := do.MustInvokeNamed[string](i, "server.webroot")

	return newTemplates(webroot), nil
}

func (t templates) executeTemplate(wr io.Writer, name string, data any) error {
	err := t[name].ExecuteTemplate(wr, name, data)
	if err != nil {
		return fmt.Errorf("execute template %q error: %w", name, err)
	}

	return nil
}
