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
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// Validation errors.
var (
	ErrInvalidUser    = aerr.New("invalid username").WithTag(aerr.ValidationError)
	ErrInvalidDevice  = aerr.New("invalid device").WithTag(aerr.ValidationError)
	ErrInvalidPodcast = aerr.New("invalid podcast").WithTag(aerr.ValidationError)
	ErrInvalidEpisode = aerr.New("invalid episode").WithTag(aerr.ValidationError)
)
