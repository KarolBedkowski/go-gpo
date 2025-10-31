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
	"gitlab.com/kabes/go-gpo/internal/service"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type WEB struct {
	i do.Injector

	template templates
	webroot  string
}

func New(i do.Injector, webroot string) WEB {
	return WEB{
		i:        i,
		webroot:  webroot,
		template: newTemplates(webroot),
	}
}

func (w *WEB) Routes() chi.Router {
	deviceSrv := do.MustInvoke[*service.Device](w.i)
	// subSrv := do.MustInvoke[*service.Subs](w.i)
	usersSrv := do.MustInvoke[*service.Users](w.i)
	episodesSrv := do.MustInvoke[*service.Episodes](w.i)
	// settingsSrv := do.MustInvoke[*service.Settings](w.i)
	podcastsSrv := do.MustInvoke[*service.Podcasts](w.i)

	router := chi.NewRouter()

	router.Get("/", internal.Wrap(w.indexPage))
	router.Mount("/device", (&devicePages{deviceSrv, w.template}).Routes())
	router.Mount("/podcast", (&podcastPages{podcastsSrv, w.template}).Routes())
	router.Mount("/episode", (&episodePages{episodesSrv, w.template}).Routes())
	router.Mount("/user", (&usersPages{usersSrv, w.template}).Routes())

	fs := http.FileServerFS(staticFS)
	router.Method("GET", "/static/*", http.StripPrefix("/web/", fs))

	return router
}

func (w *WEB) indexPage(ctx context.Context, writer http.ResponseWriter, r *http.Request, logger *zerolog.Logger) {
	_ = ctx

	if err := w.template.executeTemplate(writer, "index.tmpl", nil); err != nil {
		logger.Error().Err(err).Msg("execute template error")
		internal.WriteError(writer, r, http.StatusInternalServerError, nil)
	}
}

//-----------------------------------------------

type templates map[string]*template.Template

// newTemplate loads templates.
func newTemplates(webroot string) templates {
	logger := log.Logger
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

func (t templates) executeTemplate(wr io.Writer, name string, data any) error {
	err := t[name].ExecuteTemplate(wr, name, data)
	if err != nil {
		return fmt.Errorf("execute template %q error: %w", name, err)
	}

	return nil
}
