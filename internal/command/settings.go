package command

//
// settings.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type ChangeSettingsCmd struct {
	UserName   string
	Scope      string
	DeviceName string
	Episode    string
	Podcast    string
	Set        map[string]string
	Remove     []string
}

// NewSetFavoriteEpisodeCmd return ChangeSettingsCmd for set episode as favorite.
func NewSetFavoriteEpisodeCmd(username, podcast, episode string) ChangeSettingsCmd {
	return ChangeSettingsCmd{
		UserName:   username,
		Scope:      "episode",
		DeviceName: "",
		Episode:    episode,
		Podcast:    podcast,
		Set:        map[string]string{"is_favorite": "true"},
		Remove:     nil,
	}
}

func (c *ChangeSettingsCmd) Validate() error {
	if !validators.IsValidUserName(c.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username").WithMeta("cmd", c)
	}

	switch c.Scope {
	case "account":
		// no extra check
	case "device":
		if !validators.IsValidDevName(c.DeviceName) {
			return common.ErrInvalidDevice.WithMeta("cmd", c)
		}
	case "episode":
		if c.Episode == "" {
			return common.ErrInvalidEpisode.WithUserMsg("episode can't be empty").WithMeta("cmd", c)
		}

		fallthrough
	case "podcast":
		if c.Podcast == "" {
			return common.ErrInvalidPodcast.WithUserMsg("podcast can't be empty").WithMeta("cmd", c)
		}
	default:
		return aerr.ErrValidation.WithMsg("invalid scope").WithMeta("cmd", c)
	}

	return nil
}

func (c *ChangeSettingsCmd) CombinedSetting() map[string]string {
	// its ok to update Set
	if c.Set == nil {
		c.Set = make(map[string]string)
	}

	for _, k := range c.Remove {
		c.Set[k] = ""
	}

	return c.Set
}

func (c *ChangeSettingsCmd) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", c.UserName).
		Str("scope", c.Scope).
		Str("device", c.DeviceName).
		Str("podcast", c.Podcast).
		Str("episode", c.Episode).
		Interface("set", c.Set).
		Strs("remove", c.Remove)
}
