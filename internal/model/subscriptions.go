//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import (
	"net/url"
	"strings"
	"time"
)

// ------------------------------------------------------

type Subscription struct {
	Device    string
	Podcast   string
	Action    string
	UpdatedAt time.Time
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
