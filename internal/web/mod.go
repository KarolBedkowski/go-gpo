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
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type WEB struct {
	deviceSrv   *service.Device
	subSrv      *service.Subs
	usersSrv    *service.Users
	episodesSrv *service.Episodes
	settingsSrv *service.Settings
	podcastsSrv *service.Podcasts

	template templates
	webroot  string
}

func New(
	deviceSrv *service.Device,
	subSrv *service.Subs,
	usersSrv *service.Users,
	episodesSrv *service.Episodes,
	settingsSrv *service.Settings,
	podcastsSrv *service.Podcasts,
	webroot string,
) WEB {
	return WEB{
		deviceSrv:   deviceSrv,
		subSrv:      subSrv,
		usersSrv:    usersSrv,
		episodesSrv: episodesSrv,
		settingsSrv: settingsSrv,
		podcastsSrv: podcastsSrv,
		webroot:     webroot,

		template: newTemplates(webroot),
	}
}

func (w *WEB) Routes() chi.Router {
	router := chi.NewRouter()

	router.Get("/", internal.Wrap(w.indexPage))
	router.Mount("/device", (&devicePages{w.deviceSrv, w.template}).Routes())
	router.Mount("/podcast", (&podcastPages{w.podcastsSrv, w.template}).Routes())
	router.Mount("/episode", (&episodePages{w.episodesSrv, w.template}).Routes())
	router.Mount("/user", (&usersPages{w.usersSrv, w.template}).Routes())

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
