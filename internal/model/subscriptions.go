//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import "time"

const (
	ActionUnsubscribe = "unsubscribe"
	ActionSubscribe   = "subscribe"
)

type PodcastDB struct {
	ID         int       `db:"id"`
	UserID     int       `db:"user_id"`
	Title      string    `db:"title"`
	URL        string    `db:"url"`
	Subscribed bool      `db:"subscribed"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type Subscription struct {
	Device    string
	Podcast   string
	Action    string
	UpdatedAt time.Time
}

type SubscribedPodcastDB struct {
	SubscriptionID int    `db:"subscription_id"`
	PodcastID      int    `db:"podcast_id"`
	PodcastURL     string `db:"podcast_url"`
}

type SubscribedPodcastsDB []*SubscribedPodcastDB

func (s SubscribedPodcastsDB) FindPodcastByURL(url string) *SubscribedPodcastDB {
	for _, sp := range s {
		if sp.PodcastURL == url {
			return sp
		}
	}

	return nil
}

type EpisodeDB struct {
	ID        int       `db:"id"`
	PodcastID int       `db:"user_id"`
	Title     string    `db:"title"`
	URL       string    `db:"url"`
	Action    string    `db:"action"`
	Started   int       `db:"started"`
	Position  int       `db:"position"`
	Total     int       `db:"total"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
