package command

import "gitlab.com/kabes/go-gpo/internal/aerr"

//
// settings.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

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
	if c.UserName == "" {
		return aerr.ErrValidation.WithMsg("username can't be empty")
	}

	switch c.Scope {
	case "account":
		// no extra check
	case "device":
		if c.DeviceName == "" {
			return aerr.ErrValidation.WithMsg("device can't be empty")
		}
	case "episode":
		if c.Episode == "" {
			return aerr.ErrValidation.WithMsg("episode can't be empty")
		}

		fallthrough
	case "podcast":
		if c.Podcast == "" {
			return aerr.ErrValidation.WithMsg("podcast can't be empty")
		}
	default:
		return aerr.ErrValidation.WithMsg("invalid scope")
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
