package web

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type WEB struct {
	router *chi.Mux
}

func New(i do.Injector) (WEB, error) {
	indexPage := do.MustInvoke[indexPage](i)
	devicePages := do.MustInvoke[devicePages](i)
	userPages := do.MustInvoke[userPages](i)
	episodePages := do.MustInvoke[episodePages](i)
	podcastPages := do.MustInvoke[podcastPages](i)

	router := chi.NewRouter()

	router.Mount("/", indexPage.Routes())
	router.Mount("/device", devicePages.Routes())
	router.Mount("/podcast", podcastPages.Routes())
	router.Mount("/episode", episodePages.Routes())
	router.Mount("/user", userPages.Routes())

	fs := http.FileServerFS(staticFS)
	router.Method("GET", "/static/*", http.StripPrefix("/web/", fs))

	return WEB{router: router}, nil
}

func (w *WEB) Routes() *chi.Mux {
	return w.router
}

//-----------------------------------------------

type templates map[string]*template.Template

// newTemplate loads templates.
func newTemplates(webroot string) templates {
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

		log.Logger.Debug().Msgf("loading template: %s", name)

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
