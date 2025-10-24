//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import (
	"time"

	"github.com/rs/zerolog"
)

const (
	ActionUnsubscribe = "unsubscribe"
	ActionSubscribe   = "subscribe"
)

type Podcast struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Subscribers int    `json:"subscribers"`
	LogoURL     string `json:"logo_url"`
	Website     string `json:"website"`
	MygpoLink   string `json:"mygpo_link"`
}

type Subscription struct {
	Device    string
	Podcast   string
	Action    string
	UpdatedAt time.Time
}

type Episode struct {
	Podcast   string
	Episode   string
	Device    string
	Action    string
	Timestamp time.Time
	Started   *int
	Position  *int
	Total     *int
}

func (e Episode) MarshalZerologObject(event *zerolog.Event) {
	event.Str("podcast", e.Podcast).
		Str("episode", e.Episode).
		Str("device", e.Device).
		Str("action", e.Action).
		Time("timestamp", e.Timestamp)

	if e.Started != nil {
		event.Int("started", *e.Started)
	}

	if e.Position != nil {
		event.Int("position", *e.Position)
	}

	if e.Total != nil {
		event.Int("total", *e.Total)
	}
}

type EpisodeUpdate struct {
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	PodcastTitle string    `json:"podcast_title"`
	PodcastURL   string    `json:"podcast_url"`
	Website      string    `json:"website"`
	MygpoLink    string    `json:"mygpo_link"`
	Released     time.Time `json:"released"`
	Status       string    `json:"status"`
}
