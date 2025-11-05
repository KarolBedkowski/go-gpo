package cmd

//
// errors.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import "gitlab.com/kabes/go-gpo/internal/aerr"

var ErrValidation = aerr.NewSimple("validation error").WithTag(aerr.ValidationError)
