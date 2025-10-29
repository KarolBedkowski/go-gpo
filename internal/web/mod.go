package web

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"embed"
	"html/template"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//go:embed templates/*
var content embed.FS

type WEB struct {
	deviceSrv   *service.Device
	subSrv      *service.Subs
	usersSrv    *service.Users
	episodesSrv *service.Episodes
	settingsSrv *service.Settings

	template *template.Template
}

func New(
	deviceSrv *service.Device,
	subSrv *service.Subs,
	usersSrv *service.Users,
	episodesSrv *service.Episodes,
	settingsSrv *service.Settings,
) WEB {
	t := loadTemplates()

	return WEB{
		deviceSrv:   deviceSrv,
		subSrv:      subSrv,
		usersSrv:    usersSrv,
		episodesSrv: episodesSrv,
		settingsSrv: settingsSrv,

		template: t,
	}
}

func (w *WEB) Routes() chi.Router {
	router := chi.NewRouter()

	router.Mount("/device", (&devicePage{w.deviceSrv, w.template}).Routes())

	return router
}

// loadTemplate loads templates.
func loadTemplates() *template.Template {
	tmpl := template.New("")
	logger := log.Logger

	direntries, err := content.ReadDir("templates")
	if err != nil {
		panic(err)
	}

	for _, de := range direntries {
		if de.IsDir() {
			continue
		}

		logger.Debug().Msgf("loading template: %s", de.Name())

		tmpl, err = tmpl.New(de.Name()).ParseFS(content, "templates/"+de.Name())
		if err != nil {
			panic(err)
		}
	}

	return tmpl
}
