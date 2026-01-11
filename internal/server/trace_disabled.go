//go:build !trace

package server

//
// trace_disabled.go
// Copyright (C) 2026 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/kabes/go-gpo/internal/config"
)

func newTracingMiddleware(cfg *config.ServerConf) func(http.Handler) http.Handler {
	_ = cfg

	return func(next http.Handler) http.Handler {
		return next
	}
}

func mountXTrace(group chi.Router, webroot string) {}

//-------------------------------------------------------------

func newFRMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}
