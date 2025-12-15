package command

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type AddActionCmd struct {
	UserName string
	Actions  []model.Episode
}

func (u *AddActionCmd) Validate() error {
	if !validators.IsValidUserName(u.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username").WithMeta("cmd", u)
	}

	if len(u.Actions) == 0 {
		return aerr.ErrValidation.WithUserMsg("no actions to add").WithMeta("cmd", u)
	}

	for _, action := range u.Actions {
		if action.URL == "" {
			return aerr.ErrValidation.WithUserMsg("invalid (empty) action episode url").WithMeta("cmd", u)
		}

		if action.Podcast == nil {
			return aerr.ErrValidation.WithUserMsg("invalid (empty) podcast in action").WithMeta("cmd", u)
		}

		if action.Podcast.URL == "" {
			return common.ErrInvalidPodcast.WithUserMsg("invalid (empty) action podcast url").WithMeta("cmd", u)
		}

		if !validators.IsValidEpisodeAction(action.Action) {
			return aerr.ErrValidation.WithUserMsg("invalid action").WithMeta("action", action.Action, "cmd", u)
		}
	}

	return nil
}
