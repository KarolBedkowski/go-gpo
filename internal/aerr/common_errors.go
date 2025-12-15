package aerr

import (
	"github.com/rs/zerolog"
)

// common_errors.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

const (
	InternalError      = "internal error"
	ValidationError    = "validation error"
	DataError          = "data error"
	ConfigurationError = "configuration error"
)

var (
	ErrValidation  = New("validation error").WithTag(ValidationError)
	ErrInvalidConf = New("invalid configuration").WithTag(ConfigurationError)
	ErrDatabase    = New("database error").WithTag(InternalError).WithUserMsg("database error")
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
