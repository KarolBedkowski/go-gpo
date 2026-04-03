// Package web handle request to /web endpoint.
package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

func urlParamAsInt(r *http.Request, param string) (int64, error) {
	ids := chi.URLParam(r, param)
	if ids == "" {
		return 0, aerr.ErrBadRequest.WithMsg("Invalid %s.", param)
	}

	id, err := strconv.ParseInt(ids, 10, 64)
	if err != nil {
		return 0, aerr.ErrBadRequest.WithMsg("Invalid %s.", param)
	}

	return id, nil
}

func queryParamAsInt(r *http.Request, param string) (int64, error) {
	ids := r.URL.Query().Get(param)
	if ids == "" {
		return 0, aerr.ErrBadRequest.WithMsg("Invalid %s.", param)
	}

	id, err := strconv.ParseInt(ids, 10, 64)
	if err != nil {
		return 0, aerr.ErrBadRequest.WithMsg("Invalid %s.", param)
	}

	return id, nil
}
