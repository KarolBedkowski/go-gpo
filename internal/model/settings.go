package model

import "gitlab.com/kabes/go-gpo/internal/aerr"

//
// settings.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type SettingsKey struct {
	UserName   string
	Scope      string
	DeviceName string
	Episode    string
	Podcast    string
}

func NewSettingsKey(username, scope, device, podcast, episode string) SettingsKey {
	return SettingsKey{
		UserName:   username,
		Scope:      scope,
		DeviceName: device,
		Episode:    episode,
		Podcast:    podcast,
	}
}

func (s *SettingsKey) Validate() error {
	if s.UserName == "" {
		return aerr.ErrValidation.WithMsg("username can't be empty")
	}

	switch s.Scope {
	case "account":
		// no extra check
	case "device":
		if s.DeviceName == "" {
			return aerr.ErrValidation.WithMsg("device can't be empty")
		}
	case "episode":
		if s.Episode == "" {
			return aerr.ErrValidation.WithMsg("episode can't be empty")
		}

		fallthrough
	case "podcast":
		if s.Podcast == "" {
			return aerr.ErrValidation.WithMsg("podcast can't be empty")
		}
	default:
		return aerr.ErrValidation.WithMsg("invalid scope")
	}

	return nil
}

//------------------------------------------------------------------------------

type Settings map[string]string
