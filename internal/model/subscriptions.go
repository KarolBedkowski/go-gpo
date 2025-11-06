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

// ------------------------------------------------------

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
			return aerr.ErrValidation.Clone().WithUserMsg("duplicated url: %s", i)
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
