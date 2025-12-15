package query

//
// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type GetDevicesQuery struct {
	UserName string
}

func (q *GetDevicesQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	return nil
}
