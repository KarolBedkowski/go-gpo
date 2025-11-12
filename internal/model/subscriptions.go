//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

const (
	ActionUnsubscribe = "unsubscribe"
	ActionSubscribe   = "subscribe"
)

type Podcast struct {
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string
}

type PodcastWithLastEpisode struct {
	Title       string
	URL         string
	Description string
	Subscribers int
	LogoURL     string
	Website     string
	MygpoLink   string

	LastEpisode *Episode
}

type Subscription struct {
	Device    string
	Podcast   string
	Action    string
	UpdatedAt time.Time
}

// ------------------------------------------------------

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
		Device:    episodedb.Device,
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
		Device:     e.Device,
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
		Title:        nvl(episodedb.Title, episodedb.URL),
		URL:          episodedb.URL,
		PodcastTitle: nvl(episodedb.PodcastTitle, episodedb.PodcastURL),
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
			Podcast:   nvl(episodedb.PodcastTitle, episodedb.PodcastURL),
			Episode:   nvl(episodedb.Title, episodedb.URL),
			Device:    episodedb.Device,
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

// ------------------------------------------------------

type SubscriptionChanges struct {
	Add         []string
	Remove      []string
	ChangedURLs [][]string
}

func NewSubscriptionChanges(add, remove []string) SubscriptionChanges {
	add, chAdd := SanitizeURLs(add)
	remove, chRem := SanitizeURLs(remove)

	changes := make([][]string, 0)
	changes = append(changes, chAdd...)
	changes = append(changes, chRem...)

	return SubscriptionChanges{
		Add:         add,
		Remove:      remove,
		ChangedURLs: changes,
	}
}

func (s *SubscriptionChanges) Validate() error {
	for _, i := range s.Add {
		if slices.Contains(s.Remove, i) {
			return aerr.ErrValidation.WithUserMsg("duplicated url: %s", i)
		}
	}

	return nil
}

// ------------------------------------------------------

type SubscribedURLs []string

func NewSubscribedURLS(urls []string) SubscribedURLs {
	sanitized := make([]string, 0, len(urls))

	for _, u := range urls {
		if s := sanitizeURL(u); s != "" {
			sanitized = append(sanitized, s)
		}
	}

	return SubscribedURLs(sanitized)
}

// ------------------------------------------------------

func SanitizeURLs(urls []string) ([]string, [][]string) {
	res := make([]string, 0, len(urls))
	changes := make([][]string, 0)

	for _, u := range urls {
		su := sanitizeURL(u)

		if su == "" {
			continue
		}

		if su != u {
			changes = append(changes, []string{u, su})
		}

		res = append(res, su)
	}

	return res, changes
}

func sanitizeURL(u string) string {
	su := strings.TrimSpace(u)

	url, err := url.Parse(su)
	if err != nil || (url.Scheme != "http" && url.Scheme != "https") {
		return ""
	}

	return su
}
