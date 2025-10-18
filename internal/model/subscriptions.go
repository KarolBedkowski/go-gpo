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
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Title      string    `db:"title"`
	URL        string    `db:"url"`
	Subscribed bool      `db:"subscribed"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (p *PodcastDB) Clone() *PodcastDB {
	return &PodcastDB{
		ID:         p.ID,
		UserID:     p.UserID,
		Title:      p.Title,
		URL:        p.URL,
		Subscribed: p.Subscribed,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

type PodcastsDB []*PodcastDB

func (s PodcastsDB) FindPodcastByURL(url string) *PodcastDB {
	for _, sp := range s {
		if sp.URL == url {
			return sp
		}
	}

	return nil
}

func (s PodcastsDB) ToURLs() []string {
	res := make([]string, 0, len(s))
	for _, p := range s {
		res = append(res, p.URL)
	}

	return res
}

func (s PodcastsDB) ToMap() map[string]*PodcastDB {
	res := make(map[string]*PodcastDB)

	for _, p := range s {
		res[p.URL] = p
	}

	return res
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
	ID        int64     `db:"id"`
	PodcastID int64     `db:"podcast_id"`
	Title     string    `db:"title"`
	URL       string    `db:"url"`
	Action    string    `db:"action"`
	Started   int       `db:"started"`
	Position  int       `db:"position"`
	Total     int       `db:"total"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	PodcastURL string     `db:"podcast_url"`
	Podcast    *PodcastDB `db:"-"`
}

type Episode struct {
	Podcast   string
	Episode   string
	Device    string
	Action    string
	Timestamp time.Time
	Started   int
	Position  int
	Total     int
}
