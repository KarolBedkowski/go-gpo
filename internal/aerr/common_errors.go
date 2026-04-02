package aerr

import (
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
	ErrDatabase    = New("database error").WithTag(InternalError).WithUserMsg("database error")
	ErrBadRequest  = New("bad request").WithTag(NotFound).WithUserMsg("bad request")
	ErrNoData      = New("no result").WithTag(NotFound)
)

func IsSerious(err error) bool {
	tags := GetTags(err)

	if len(tags) == 1 && tags[0] == ValidationError {
		return false
	}

	return true
}

func LogLevelForError(err error) zerolog.Level {
	if IsSerious(err) {
		return zerolog.WarnLevel
	}

	// all others are usually user errors and not required logging.
	return zerolog.DebugLevel
}
