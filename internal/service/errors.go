package service

//
// errors.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"errors"

	"gitlab.com/kabes/go-gpo/internal/aerr"
)

var ErrRepositoryError = aerr.NewSimple("database error").WithTag(aerr.InternalError).
	WithUserMsg("database error")

var (
	ErrEmptyUsername  = aerr.NewSimple("username can't be empty").WithTag(aerr.ValidationError)
	ErrUnknownDevice  = aerr.NewSimple("unknown device").WithUserMsg("unknown device").WithTag(aerr.DataError)
	ErrUnknownPodcast = errors.New("unknown podcast")
	ErrUnknownEpisode = errors.New("unknown episode")
)

var (
	ErrUnknownUser       = aerr.NewSimple("unknown user").WithTag(aerr.DataError)
	ErrUnauthorized      = aerr.New("unauthorized").WithUserMsg("authorization failed")
	ErrUserAccountLocked = aerr.New("locked account").WithUserMsg("account is locked")
	ErrUserExists        = aerr.New("username exists").WithUserMsg("user name already exists")
)
