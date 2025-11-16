package queries

//
// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import "gitlab.com/kabes/go-gpo/internal/aerr"

type QueryDevices struct {
	UserName string
}

func (q *QueryDevices) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	return nil
}
