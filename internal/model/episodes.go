package model

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type Episode struct {
	ID        int64
	Action    string
	Timestamp time.Time
	Started   *int
	Position  *int
	Total     *int
	GUID      *string
	Title     string
	URL       string

	Podcast Podcast
	Device  *Device
}

func (e *Episode) DeviceName() string {
	if e.Device == nil {
		return ""
	}

	return e.Device.Name
}

func (e *Episode) Validate() error {
	if !validators.IsValidEpisodeAction(e.Action) {
		return aerr.ErrValidation.WithUserMsg("invalid action")
	}

	if e.Action != "play" {
		if e.Started != nil || e.Position != nil || e.Total != nil {
			return aerr.ErrValidation.
				WithUserMsg("for action other than 'play' - started, position and total should be not set")
		}
	}

	return nil
}

func (e *Episode) MarshalZerologObject(event *zerolog.Event) {
	event.Interface("podcast", e.Podcast).
		Str("url", e.URL).
		Object("device", e.Device).
		Str("action", e.Action).
		Time("timestamp", e.Timestamp).
		Any("guid", e.GUID).
		Any("started", e.Started).
		Any("position", e.Position).
		Any("total", e.Total)
}

// ------------------------------------------------------

type Favorite struct {
	Title        string
	URL          string
	PodcastTitle string
	PodcastURL   string
	Website      string
	MygpoLink    string
	Released     time.Time
}

// ------------------------------------------------------

type EpisodeUpdate struct {
	Title        string
	URL          string
	PodcastTitle string
	PodcastURL   string
	Website      string
	MygpoLink    string
	Released     time.Time
	Status       string

	Episode *Episode
}

// ------------------------------------------------------

type EpisodeLastAction struct {
	PodcastTitle string
	PodcastURL   string
	Episode      string
	Device       string
	Action       string
	Timestamp    time.Time
	Started      *int
	Position     *int
	Total        *int
}
