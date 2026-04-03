package aerr

import (
	"slices"

	"github.com/rs/zerolog"
)

// common_errors.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

// Main errors categories.
const (
	InternalError      = "internal error"
	ValidationError    = "validation error"
	ConfigurationError = "configuration error"
	NotFound           = "not found"
	BadRequest         = "bad request"
)

var (
	ErrValidation  = New("validation error").WithTag(ValidationError)
	ErrInvalidConf = New("invalid configuration").WithTag(ConfigurationError)
	ErrDatabase    = New("database error").WithTag(InternalError)
	ErrBadRequest  = New("bad request").WithTag(BadRequest)
	ErrNoData      = New("no result").WithTag(NotFound)
)

func LogLevelForError(err error) zerolog.Level {
	switch tags := GetTags(err); {
	case len(tags) == 0:
		// this is unknown error
		return zerolog.WarnLevel
	case slices.Contains(tags, InternalError):
		// internal server error
		return zerolog.ErrorLevel
	default:
		// all others are usually user errors and not required logging.
		return zerolog.DebugLevel
	}
}
