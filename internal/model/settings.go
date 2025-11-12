package model

import "gitlab.com/kabes/go-gpo/internal/aerr"

//
// settings.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type SettingsKey struct {
	Username string
	Scope    string
	Device   string
	Episode  string
	Podcast  string
}

func NewSettingsKey(username, scope, device, podcast, episode string) (SettingsKey, error) {
	if username == "" {
		return SettingsKey{}, aerr.ErrValidation.WithMsg("username can't be empty")
	}

	switch scope {
	case "account":
		// no extra check
	case "device":
		if device == "" {
			return SettingsKey{}, aerr.ErrValidation.WithMsg("device can't be empty")
		}
	case "episode":
		if episode == "" {
			return SettingsKey{}, aerr.ErrValidation.WithMsg("episode can't be empty")
		}

		fallthrough
	case "podcast":
		if podcast == "" {
			return SettingsKey{}, aerr.ErrValidation.WithMsg("podcast can't be empty")
		}
	default:
		return SettingsKey{}, aerr.ErrValidation.WithMsg("invalid scope")
	}

	settkey := SettingsKey{
		Username: username,
		Scope:    scope,
		Device:   device,
		Episode:  episode,
		Podcast:  podcast,
	}

	return settkey, nil
}

//------------------------------------------------------------------------------

type Settings map[string]string
