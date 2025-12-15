package web

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
)

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
