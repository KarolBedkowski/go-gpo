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
	UserID int64

	PodcastID *int64
	EpisodeID *int64
	DeviceID  *int64
	Scope     string
	Key       string
}

func (s *SettingsKey) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("user_id", s.UserID).
		Any("podcast_id", s.PodcastID).
		Any("episode_id", s.EpisodeID).
		Any("device_id", s.DeviceID).
		Str("scope", s.Scope)
}

type UserSettings struct {
	UserID int64

	PodcastID *int64
	EpisodeID *int64
	DeviceID  *int64
	Scope     string
	Key       string
	Value     string
}

func (u *UserSettings) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("user_id", u.UserID).
		Any("podcast_id", u.PodcastID).
		Any("episode_id", u.EpisodeID).
		Any("device_id", u.DeviceID).
		Str("scope", u.Scope).
		Str("value", u.Value)
}

func (u *UserSettings) ToKey() SettingsKey {
	return SettingsKey{
		UserID:    u.UserID,
		PodcastID: u.PodcastID,
		DeviceID:  u.DeviceID,
		Scope:     u.Scope,
		Key:       u.Key,
	}
}
