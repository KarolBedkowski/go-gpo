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

	"gitlab.com/kabes/go-gpo/internal/aerr"
)

// ------------------------------------------------------

type Subscription struct {
	Device    string
	Podcast   string
	Action    string
	UpdatedAt time.Time
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
		if s := SanitizeURL(u); s != "" {
			sanitized = append(sanitized, s)
		}
	}

	return SubscribedURLs(sanitized)
}

// ------------------------------------------------------

type SubscriptionState struct {
	Added   []Podcast
	Removed []Podcast
}

func (s *SubscriptionState) AddedURLs() []string {
	return PodcastsToUrls(s.Added)
}

func (s *SubscriptionState) RemovedURLs() []string {
	return PodcastsToUrls(s.Removed)
}

// ------------------------------------------------------

// SanitizeURLs remove non-http* urls; correct others. Return list of "safe" urls and list of changes
// in url [[old url, corrected url]].
func SanitizeURLs(urls []string) ([]string, [][]string) {
	res := make([]string, 0, len(urls))
	changes := make([][]string, 0)

	for _, u := range urls {
		su := SanitizeURL(u)

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

func SanitizeURL(u string) string {
	su := strings.TrimSpace(u)

	url, err := url.Parse(su)
	if err != nil || (url.Scheme != "http" && url.Scheme != "https") {
		return ""
	}

	return su
}
