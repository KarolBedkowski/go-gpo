//go:build !trace

package server

//
// tracing.go
// Copyright (C) 2026 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func newTracingMiddleware(cfg *Configuration) func(http.Handler) http.Handler {
	_ = cfg

	return func(next http.Handler) http.Handler {
		return next
	}
}

func mountXTrace(group chi.Router, webroot string) {
}

//-------------------------------------------------------------

func newFRMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}
