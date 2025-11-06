package aerr

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
	ErrValidation  = NewSimple("validation error").WithTag(ValidationError)
	ErrInvalidConf = NewSimple("invalid configuration").WithTag(ConfigurationError)
	ErrDatabase    = NewSimple("database error").WithTag(InternalError).WithUserMsg("database error")
)
