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
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type AddActionCmd struct {
	UserName string
	Actions  []model.Episode
}

func (u *AddActionCmd) Validate() error {
	if !validators.IsValidUserName(u.UserName) {
		return aerr.ErrValidation.WithUserMsg("invalid username").WithMeta("cmd", u)
	}

	if len(u.Actions) == 0 {
		return aerr.ErrValidation.WithMsg("no actions to add").WithMeta("cmd", u)
	}

	for _, a := range u.Actions {
		if !validators.IsValidEpisodeAction(a.Action) {
			return aerr.ErrValidation.WithMsg("invalid action").WithMeta("action", a.Action, "cmd", u)
		}
	}

	return nil
}
