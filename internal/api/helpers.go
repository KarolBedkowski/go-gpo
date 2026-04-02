package api

// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// getSinceParameter from request url query.
func getSinceParameter(r *http.Request) (time.Time, error) {
	since := time.Time{}

	if s := r.URL.Query().Get("since"); s != "" {
		se, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return since, fmt.Errorf("parse error: %w", err)
		}

		since = time.Unix(se, 0).UTC()
	}

	return since, nil
}

// checkAndWriteError decode and write error to ResponseWriter.
func renderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case aerr.HasTag(err, aerr.InternalError):
		writeError(w, r, http.StatusInternalServerError, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.NotFound):
		writeError(w, r, http.StatusNotFound, aerr.GetUserMessage(err))
	case aerr.HasTag(err, aerr.ValidationError):
		writeError(w, r, http.StatusBadRequest, aerr.GetUserMessage(err))
	default:
		writeError(w, r, http.StatusInternalServerError, aerr.GetUserMessage(err))
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, details string) {
	var reqid, now string

	if status == http.StatusInternalServerError {
		rid, _ := hlog.IDFromRequest(r)
		reqid = rid.String()
		now = time.Now().Format(time.RFC3339Nano)
	}

	msg := http.StatusText(status)

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		res := struct {
			Error   string `json:"error"`
			ReqID   string `json:"request_id,omitempty"`
			TS      string `json:"ts,omitempty"`
			Details string `json:"details,omitempty"`
		}{msg, reqid, now, details}

		render.Status(r, status)
		render.JSON(w, r, &res)

		return
	}

	http.Error(w, msg, status)

	if details != "" {
		fmt.Fprintln(w, details)
	}

	if reqid != "" {
		fmt.Fprintln(w, "reqid="+reqid) //nolint:gosec
	}

	if now != "" {
		fmt.Fprintln(w, "ts="+now)
	}
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
