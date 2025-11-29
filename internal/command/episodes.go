package command

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
)

type AddActionCmd struct {
	UserName string
	Actions  []model.Episode
}

func (u *AddActionCmd) Validate() error {
	if u.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	if len(u.Actions) == 0 {
		return aerr.ErrValidation.WithMsg("no actions to add")
	}

	return nil
}
