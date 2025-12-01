package model

import "github.com/rs/zerolog"

//
// settings.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

//------------------------------------------------------------------------------

type Settings map[string]string

type SettingsKey struct {
	PodcastID *int32
	EpisodeID *int32
	DeviceID  *int32
	Scope     string
	Key       string
	UserID    int32
}

func (s *SettingsKey) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("user_id", s.UserID).
		Any("podcast_id", s.PodcastID).
		Any("episode_id", s.EpisodeID).
		Any("device_id", s.DeviceID).
		Str("scope", s.Scope)
}
