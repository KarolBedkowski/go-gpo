// model.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package repository

import (
	"errors"
	"time"

	"github.com/rs/zerolog"
)

var ErrNoData = errors.New("no result")

type DeviceDB struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	DevType   string    `db:"dev_type"`
	Caption   string    `db:"caption"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Subscriptions int `db:"subscriptions"`
}

func (d DeviceDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", d.ID).
		Int64("user_id", d.UserID).
		Str("name", d.Name).
		Str("type", d.DevType).
		Str("caption", d.Caption).
		Time("created_at", d.CreatedAt).
		Time("updated_at", d.UpdatedAt).
		Int("subscriptions", d.Subscriptions)
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

func (p PodcastDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", p.ID).
		Int64("user_id", p.UserID).
		Str("title", p.Title).
		Str("url", p.URL).
		Bool("subscribed", p.Subscribed).
		Time("created_at", p.CreatedAt).
		Time("updated_at", p.UpdatedAt)
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

func (e EpisodeDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", e.ID).
		Int64("podcast_id", e.PodcastID).
		Int64("device_id", e.DeviceID).
		Str("title", e.Title).
		Str("url", e.URL).
		Str("action", e.Action)

	if e.Started != nil {
		event.Int("started", *e.Started)
	}

	if e.Position != nil {
		event.Int("position", *e.Position)
	}

	if e.Total != nil {
		event.Int("total", *e.Total)
	}

	event.Time("created_at", e.CreatedAt).
		Time("updated_at", e.UpdatedAt)

	event.Dict("podcast", zerolog.Dict().
		Str("podcast_url", e.PodcastURL).
		Str("podcast_title", e.PodcastTitle))

	event.Str("device", e.Device)
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

func (u UserDB) MarshalZerologObject(event *zerolog.Event) {
	pass := ""
	if u.Password != "" {
		pass = "***"
	}

	event.Int64("id", u.ID).
		Str("user_name", u.Username).
		Str("Password", pass).
		Str("email", u.Email).
		Str("name", u.Name).
		Time("created_at", u.CreatedAt).
		Time("updated_at", u.UpdatedAt)
}

type SettingsDB struct {
	UserID    int64  `db:"user_id"`
	PodcastID *int64 `db:"podcast_id"`
	EpisodeID *int64 `db:"episode_id"`
	DeviceID  *int64 `db:"device_id"`
	Scope     string `db:"scope"`
	Key       string `db:"key"`
	Value     string `db:"value"`
}

func (s SettingsDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("user_id", s.UserID).
		Any("podcast_id", s.PodcastID).
		Any("episode_id", s.EpisodeID).
		Any("device_id", s.DeviceID).
		Str("scope", s.Scope).
		Str("key", s.Key).
		Str("value", s.Value)
}
