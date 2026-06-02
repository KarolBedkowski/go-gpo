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

var (
	ErrUserAccountLocked = aerr.ErrAuthentication.WithMsg("locked account").WithUserMsg("account is locked")
	ErrInvalidPassword   = aerr.ErrAuthentication.WithDetails("invalid password")
)
