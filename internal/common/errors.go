package common

//
// Common application errors
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

var (
	ErrUnauthorized      = aerr.New("unauthorized").WithUserMsg("authorization failed")
	ErrUserAccountLocked = aerr.New("locked account").WithUserMsg("account is locked")
)

// Validation errors.
var (
	ErrUnknownUser    = aerr.New("unknown user").WithTag(aerr.ValidationError)
	ErrEmptyUsername  = aerr.New("username can't be empty").WithTag(aerr.ValidationError)
	ErrUnknownDevice  = aerr.New("unknown device").WithTag(aerr.ValidationError)
	ErrUnknownPodcast = aerr.New("unknown podcast").WithTag(aerr.ValidationError)
	ErrUnknownEpisode = aerr.New("unknown episode").WithTag(aerr.ValidationError)
	ErrUserExists     = aerr.New("username exists").WithUserMsg("user name already exists")
	ErrInvalidUser    = aerr.New("invalid user").WithTag(aerr.ValidationError)
	ErrInvalidDevice  = aerr.New("invalid device").WithTag(aerr.ValidationError)
	ErrInvalidPodcast = aerr.New("invalid podcast").WithTag(aerr.ValidationError)
	ErrInvalidEpisode = aerr.New("invalid episode").WithTag(aerr.ValidationError)
)

var ErrNoData = errors.New("no result")
