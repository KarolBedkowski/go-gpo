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

// jsonpWriter wrap response with jsonp function when this function name is given in `jsonp` url parameter.
type jsonpWriter struct {
	http.ResponseWriter

	jsonp string
}

func newJSONPWriter(r *http.Request, w http.ResponseWriter) jsonpWriter {
	return jsonpWriter{w, r.URL.Query().Get("jsonp")}
}

//nolint:wrapcheck
func (j jsonpWriter) Write(buf []byte) (int, error) {
	if j.jsonp == "" {
		return j.ResponseWriter.Write(buf)
	}

	count1, err := j.ResponseWriter.Write([]byte(j.jsonp + "("))
	if err != nil {
		return 0, err
	}

	count2, err := j.ResponseWriter.Write(buf)
	if err != nil {
		return 0, err
	}

	count3, err := j.ResponseWriter.Write([]byte(")"))
	if err != nil {
		return 0, err
	}

	return count1 + count2 + count3, nil
}
