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

var ErrRepositoryError = aerr.New("database error").
	WithTag(aerr.InternalError).
	WithUserMsg("database error")

var ErrUserExists = aerr.New("username exists").WithUserMsg("user name already exists").
	WithTag(aerr.ValidationError)

const AuthenticationError = "authentication error"

var (
	ErrUnauthorized      = aerr.New("unauthorized").WithUserMsg("authorization failed").WithTag(AuthenticationError)
	ErrUserAccountLocked = aerr.New("locked account").WithUserMsg("account is locked").WithTag(AuthenticationError)
)
