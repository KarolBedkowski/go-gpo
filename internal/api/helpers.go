// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
)

// ensureList create empty list if `inp` is null or return `inp` otherwise.
// func ensureList[T any](inp []T) []T {
// 	if inp == nil {
// 		return make([]T, 0)
// 	}

// 	return inp
// }

// getSinceParameter from request url query.
func getSinceParameter(r *http.Request) (time.Time, error) {
	since := time.Time{}

	if s := r.URL.Query().Get("since"); s != "" {
		se, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return since, fmt.Errorf("parse since %q error: %w", s, err)
		}

		since = time.Unix(se, 0).UTC()
	}

	return since, nil
}

// checkAndWriteError decode and write error to ResponseWriter.
func checkAndWriteError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, common.ErrUnknownDevice):
		status = http.StatusNotFound

	case aerr.HasTag(err, aerr.InternalError):
		// write message if is defined in error
		status = http.StatusInternalServerError

	case aerr.HasTag(err, aerr.ValidationError):
		status = http.StatusBadRequest

	case aerr.HasTag(err, aerr.DataError):
		status = http.StatusBadRequest
	}

	writeError(w, r, status)
}

func writeError(w http.ResponseWriter, r *http.Request, status int) {
	msg := http.StatusText(status)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		res := struct {
			Error string `json:"error"`
		}{msg}

		render.Status(r, status)
		render.JSON(w, r, &res)

		return
	}

	http.Error(w, msg, status)
}
