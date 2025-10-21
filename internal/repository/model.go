// model.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package repository

import "time"

type DeviceDB struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	DevType   string    `db:"dev_type"`
	Caption   string    `db:"caption"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Subscriptions int `db:"-"`
}

type DevicesDB []*DeviceDB

func (d DevicesDB) ToMap() map[string]*DeviceDB {
	devices := make(map[string]*DeviceDB)

	for _, dev := range d {
		devices[dev.Name] = dev
	}

	return devices
}

func (d DevicesDB) ToIDsMap() map[string]int64 {
	devices := make(map[string]int64)

	for _, dev := range d {
		devices[dev.Name] = dev.ID
	}

	return devices
}

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

type PodcastsDB []PodcastDB

func (s PodcastsDB) FindSubscribedPodcastByURL(url string) (PodcastDB, bool) {
	for _, sp := range s {
		if sp.URL == url && sp.Subscribed {
			return sp, true
		}
	}

	return PodcastDB{}, false
}

func (s PodcastsDB) FindPodcastByURL(url string) (PodcastDB, bool) {
	for _, sp := range s {
		if sp.URL == url {
			return sp, true
		}
	}

	return PodcastDB{}, false
}

func (s PodcastsDB) ToURLs() []string {
	res := make([]string, 0, len(s))
	for _, p := range s {
		res = append(res, p.URL)
	}

	return res
}

func (s PodcastsDB) ToMap() map[string]PodcastDB {
	res := make(map[string]PodcastDB)

	for _, p := range s {
		res[p.URL] = p
	}

	return res
}

func (s PodcastsDB) ToIDsMap() map[string]int64 {
	res := make(map[string]int64)

	for _, p := range s {
		res[p.URL] = p.ID
	}

	return res
}

type EpisodeDB struct {
	ID        int64     `db:"id"`
	PodcastID int64     `db:"podcast_id"`
	DeviceID  int64     `db:"device_id"`
	Title     string    `db:"title"`
	URL       string    `db:"url"`
	Action    string    `db:"action"`
	Started   *int      `db:"started"`
	Position  *int      `db:"position"`
	Total     *int      `db:"total"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	PodcastURL   string `db:"podcast_url"`
	PodcastTitle string `db:"podcast_title"`
	Device       string `db:"device_name"`
}

type UserDB struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type UserAlias struct {
	UserID   int64  `db:"user_id"`
	Username string `db:"username"`
}
