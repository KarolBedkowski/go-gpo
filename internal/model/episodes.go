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
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

const (
	ActionPlay string = "play"
	ActionNew  string = "new"
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

	if e.Action != ActionPlay {
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

func NewFavoriteFromModel(episodedb *Episode) Favorite {
	return Favorite{
		Title:        common.Coalesce(episodedb.Title, episodedb.URL),
		URL:          episodedb.URL,
		PodcastTitle: common.Coalesce(episodedb.Podcast.Title, episodedb.Podcast.URL),
		PodcastURL:   episodedb.Podcast.URL,
		Website:      "",
		MygpoLink:    "",
		Released:     episodedb.Timestamp, // FIXME: this is not release date...
	}
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

// NewUpisodeUpdateFromModel create new EpisodeUpdate WITHOUT Episode.
func NewEpisodeUpdate(episodedb *Episode) EpisodeUpdate {
	return EpisodeUpdate{
		Title:        episodedb.Title,
		URL:          episodedb.URL,
		PodcastTitle: episodedb.Podcast.Title,
		PodcastURL:   episodedb.Podcast.URL,
		Status:       episodedb.Action,
		// do not tracking released time; use updated time
		Released:  episodedb.Timestamp,
		Episode:   nil,
		Website:   "",
		MygpoLink: "",
	}
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

func NewEpisodeLastAction(episodedb *Episode) EpisodeLastAction {
	dev := ""
	if episodedb.Device != nil {
		dev = episodedb.Device.Name
	}

	episode := EpisodeLastAction{
		PodcastURL:   episodedb.Podcast.URL,
		PodcastTitle: episodedb.Podcast.Title,
		Device:       dev,
		Episode:      episodedb.URL,
		Action:       episodedb.Action,
		Timestamp:    episodedb.Timestamp,
		Started:      nil,
		Position:     nil,
		Total:        nil,
	}
	if episodedb.Action == ActionPlay {
		episode.Started = episodedb.Started
		episode.Position = episodedb.Position
		episode.Total = episodedb.Total
	}

	return episode
}

// NewEpisodeUpdateWithEpisode create new EpisodeUpdate WITH Episode.
func NewEpisodeUpdateWithEpisode(episodedb *Episode) EpisodeUpdate {
	episodeUpdate := EpisodeUpdate{
		Title:        episodedb.Title,
		URL:          episodedb.URL,
		PodcastTitle: episodedb.Podcast.Title,
		PodcastURL:   episodedb.Podcast.URL,
		Status:       episodedb.Action,
		// do not tracking released time; use updated time
		Released:  episodedb.Timestamp,
		Episode:   nil,
		Website:   "",
		MygpoLink: "",
	}

	if episodedb.Action != ActionNew {
		episodeUpdate.Episode = &Episode{
			Podcast:   episodedb.Podcast,
			URL:       common.Coalesce(episodedb.Title, episodedb.URL),
			Device:    episodedb.Device,
			Action:    episodedb.Action,
			Timestamp: episodedb.Timestamp,
			GUID:      episodedb.GUID,
			Started:   nil,
			Position:  nil,
			Total:     nil,
		}
		if episodedb.Action == ActionPlay {
			episodeUpdate.Episode.Started = episodedb.Started
			episodeUpdate.Episode.Position = episodedb.Position
			episodeUpdate.Episode.Total = episodedb.Total
		}
	}

	return episodeUpdate
}
