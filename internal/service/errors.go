package service

//
// errors.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

var (
	ErrUnknownUser   = aerr.NewSimple("unknown user").WithTag(aerr.DataError)
	ErrUnknownDevice = aerr.NewSimple("unknown device").WithUserMsg("unknown device").WithTag(aerr.DataError)
	ErrInvalidData   = aerr.NewSimple("invalid data").WithTag(aerr.DataError)

	ErrRepositoryError = aerr.NewSimple("database error").WithTag(aerr.InternalError).
				WithUserMsg("database error")
)
