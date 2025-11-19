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
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type Episode struct {
	Podcast   string
	Episode   string
	Device    string
	Action    string
	Timestamp time.Time
	Started   *int
	Position  *int
	Total     *int
	GUID      *string
}

func NewEpisodeFromDBModel(episodedb *repository.EpisodeDB) Episode {
	episode := Episode{
		Podcast:   episodedb.PodcastURL,
		Device:    common.NVL(episodedb.Device, ""),
		Episode:   episodedb.URL,
		Action:    episodedb.Action,
		Timestamp: episodedb.UpdatedAt,
		GUID:      episodedb.GUID,
		Started:   nil,
		Position:  nil,
		Total:     nil,
	}
	if episodedb.Action == "play" { //nolint:goconst
		episode.Started = episodedb.Started
		episode.Position = episodedb.Position
		episode.Total = episodedb.Total
	}

	return episode
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

func (e *Episode) ToDBModel() repository.EpisodeDB {
	return repository.EpisodeDB{ //nolint:exhaustruct
		URL:        e.Episode,
		Device:     common.NilIf(e.Device, ""),
		Action:     e.Action,
		UpdatedAt:  e.Timestamp,
		CreatedAt:  e.Timestamp,
		Started:    e.Started,
		Position:   e.Position,
		Total:      e.Total,
		PodcastURL: e.Podcast,
		GUID:       e.GUID,
	}
}

func (e *Episode) MarshalZerologObject(event *zerolog.Event) {
	event.Str("podcast", e.Podcast).
		Str("episode", e.Episode).
		Str("device", e.Device).
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

func NewFavoriteFromDBModel(episodedb *repository.EpisodeDB) Favorite {
	return Favorite{
		Title:        common.Coalesce(episodedb.Title, episodedb.URL),
		URL:          episodedb.URL,
		PodcastTitle: common.Coalesce(episodedb.PodcastTitle, episodedb.PodcastURL),
		PodcastURL:   episodedb.PodcastURL,
		Website:      "",
		MygpoLink:    "",
		Released:     episodedb.CreatedAt, // FIXME: this is not release date...
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

// NewEpisodeUpdateFromDBModel create new EpisodeUpdate WITHOUT Episode.
func NewEpisodeUpdateFromDBModel(episodedb *repository.EpisodeDB) EpisodeUpdate {
	return EpisodeUpdate{
		Title:        episodedb.Title,
		URL:          episodedb.URL,
		PodcastTitle: episodedb.PodcastTitle,
		PodcastURL:   episodedb.PodcastURL,
		Status:       episodedb.Action,
		// do not tracking released time; use updated time
		Released:  episodedb.UpdatedAt,
		Episode:   nil,
		Website:   "",
		MygpoLink: "",
	}
}

// NewEpisodeUpdateWithEpisodeFromDBModel create new EpisodeUpdate WITH Episode.
func NewEpisodeUpdateWithEpisodeFromDBModel(episodedb *repository.EpisodeDB) EpisodeUpdate {
	episodeUpdate := EpisodeUpdate{
		Title:        episodedb.Title,
		URL:          episodedb.URL,
		PodcastTitle: episodedb.PodcastTitle,
		PodcastURL:   episodedb.PodcastURL,
		Status:       episodedb.Action,
		// do not tracking released time; use updated time
		Released:  episodedb.UpdatedAt,
		Episode:   nil,
		Website:   "",
		MygpoLink: "",
	}

	if episodedb.Action != "new" {
		episodeUpdate.Episode = &Episode{
			Podcast:   common.Coalesce(episodedb.PodcastTitle, episodedb.PodcastURL),
			Episode:   common.Coalesce(episodedb.Title, episodedb.URL),
			Device:    common.NVL(episodedb.Device, ""),
			Action:    episodedb.Action,
			Timestamp: episodedb.UpdatedAt,
			GUID:      episodedb.GUID,
			Started:   nil,
			Position:  nil,
			Total:     nil,
		}
		if episodedb.Action == "play" {
			episodeUpdate.Episode.Started = episodedb.Started
			episodeUpdate.Episode.Position = episodedb.Position
			episodeUpdate.Episode.Total = episodedb.Total
		}
	}

	return episodeUpdate
}
