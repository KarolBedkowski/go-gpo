package api

// helpers.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// getSinceParameter from request url query.
func getSinceParameter(r *http.Request) (time.Time, error) {
	since := time.Time{}

	if s := r.URL.Query().Get("since"); s != "" {
		se, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return since, aerr.Wrapf(err, "parse since parameter error").WithMeta("since", s).
				WithTag(aerr.ValidationError)
		}

		since = time.Unix(se, 0).UTC()
	}

	return since, nil
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
