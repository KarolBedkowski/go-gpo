//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import (
	"time"
)

// ------------------------------------------------------

type Subscription struct {
	UpdatedAt time.Time
	Device    string
	Podcast   string
	Action    string
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
