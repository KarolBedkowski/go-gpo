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
	InternalError       = Tag("internal error")
	ValidationError     = Tag("validation error")
	ConfigurationError  = Tag("configuration error")
	NotFound            = Tag("not found")
	BadRequest          = Tag("bad request")
	AuthenticationError = Tag("authentication error")
	AuthorizationError  = Tag("authentication error")
)

var (
	ErrValidation     = New("validation error").WithTag(ValidationError)
	ErrInvalidConf    = New("invalid configuration").WithTag(ConfigurationError)
	ErrDatabase       = New("database error").WithTag(InternalError)
	ErrBadRequest     = New("bad request").WithTag(BadRequest)
	ErrNoData         = New("no result").WithTag(NotFound)
	ErrUnauthorized   = New("unauthorized").WithTag(AuthorizationError)
	ErrAuthentication = New("authentication error").WithTag(AuthenticationError)
)

// LogLevelForError return zerolog.Lever that should be used for given error.
// Map internal errors as Error, user faults as debug and other as info.
func LogLevelForError(err error) zerolog.Level {
	switch tags := GetTags(err); {
	case len(tags) == 0:
		// this is unknown error
		return zerolog.WarnLevel
	case slices.Contains(tags, InternalError):
		// internal server error
		return zerolog.ErrorLevel
	case slices.Contains(tags, AuthenticationError):
		return zerolog.InfoLevel
	case slices.Contains(tags, AuthorizationError):
		return zerolog.InfoLevel
	default:
		// all others are usually user errors and not required logging.
		return zerolog.DebugLevel
	}
}
