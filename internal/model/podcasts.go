package model

import (
	"strings"
	"time"

	"github.com/rs/zerolog"
)

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type Podcast struct {
	ID int64

	UpdatedAt     time.Time
	MetaUpdatedAt time.Time
	Title         string
	URL           string
	Description   string
	LogoURL       string
	Website       string
	MygpoLink     string
	User          User
	Subscribers   int
	Subscribed    bool
}

func (p *Podcast) SetSubscribed(timestamp time.Time) bool {
	if p.Subscribed {
		return false
	}

	p.Subscribed = true
	p.UpdatedAt = timestamp

	return true
}

func (p *Podcast) SetUnsubscribed(timestamp time.Time) bool {
	if !p.Subscribed {
		return false
	}

	p.Subscribed = false
	p.UpdatedAt = timestamp

	return true
}

func (p *Podcast) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", p.ID).
		Str("title", p.Title).
		Str("url", p.URL).
		Str("website", p.Website).
		Str("description", p.Description).
		Bool("subscribed", p.Subscribed).
		Time("updated_at", p.UpdatedAt)
}

type PodcastWithLastEpisode struct {
	LastEpisode *Episode
	Title       string
	URL         string
	Description string
	LogoURL     string
	Website     string
	MygpoLink   string
	PodcastID   int64
	Subscribers int
	Subscribed  bool
}

func PodcastsToUrls(podcasts []Podcast) []string {
	urls := make([]string, len(podcasts))
	for i, p := range podcasts {
		urls[i] = p.URL
	}

	return urls
}

//------------------------------------------------------------------------------

type Podcasts []Podcast

func (s Podcasts) FindSubscribedPodcastByURL(url string) (Podcast, bool) {
	for _, sp := range s {
		if sp.URL == url && sp.Subscribed {
			return sp, true
		}
	}

	return Podcast{}, false
}

// FindPodcastByURL look for url in podcasts. if podcasts url or given url
// url has suffix '/' - try to match it also without it.
func (s Podcasts) FindPodcastByURL(url string) (Podcast, bool) {
	alt := ""
	if a, ok := strings.CutSuffix(url, "/"); ok {
		alt = a
	}

	for _, sp := range s {
		if sp.URL == url || sp.URL == alt {
			return sp, true
		}

		trimmed := strings.TrimSuffix(sp.URL, "/")
		if trimmed == url || trimmed == alt {
			return sp, true
		}
	}

	return Podcast{}, false
}

func (s Podcasts) ToURLs() []string {
	res := make([]string, 0, len(s))
	for _, p := range s {
		res = append(res, p.URL)
	}

	return res
}

func (s Podcasts) ToMap() map[string]Podcast {
	res := make(map[string]Podcast)

	for _, p := range s {
		res[p.URL] = p
	}

	return res
}

func (s Podcasts) ToIDsMap() map[string]int64 {
	res := make(map[string]int64)

	for _, p := range s {
		res[p.URL] = p.ID
	}

	return res
}

//------------------------------------------------------------------------------

type PodcastMetaUpdate struct {
	MetaUpdatedAt time.Time
	Title         string
	URL           string
	Description   string
	Website       string
}

type PodcastToUpdate struct {
	MetaUpdatedAt time.Time
	URL           string
}
