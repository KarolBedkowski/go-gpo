package sqlite

// model.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/model"
)

//----------------------------------------

type DeviceDB struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Name       string    `db:"name"`
	DevType    string    `db:"dev_type"`
	Caption    string    `db:"caption"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	LastSeenAt time.Time `db:"last_seen_at"`

	Subscriptions int `db:"subscriptions"`

	UserName     string `db:"user_name"`
	UserUserName string `db:"user_username"`
}

func (d *DeviceDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", d.ID).
		Int64("user_id", d.UserID).
		Str("name", d.Name).
		Str("type", d.DevType).
		Str("caption", d.Caption).
		Time("created_at", d.CreatedAt).
		Time("updated_at", d.UpdatedAt).
		Int("subscriptions", d.Subscriptions).
		Str("user_name", d.UserName).
		Str("user_username", d.UserUserName)
}

func (d *DeviceDB) toModel() *model.Device {
	return &model.Device{
		ID:            d.ID,
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
		UpdatedAt:     d.UpdatedAt,
		LastSeenAt:    d.LastSeenAt,
		User: &model.User{
			ID:       d.UserID,
			Name:     d.UserName,
			UserName: d.UserUserName,
		},
	}
}

//------------------------------------------------------------------------------

func devicesFromDb(devices []DeviceDB) []model.Device {
	res := make([]model.Device, len(devices))
	for i, r := range devices {
		res[i] = *r.toModel()
	}

	return res
}

//------------------------------------------------------------------------------

type PodcastDB struct {
	ID          int64  `db:"id"`
	UserID      int64  `db:"user_id"`
	Title       string `db:"title"`
	URL         string `db:"url"`
	Subscribed  bool   `db:"subscribed"`
	Description string `db:"description"`
	Website     string `db:"website"`

	CreatedAt     time.Time    `db:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at"`
	MetaUpdatedAt sql.NullTime `db:"metadata_updated_at"`
}

func (p *PodcastDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", p.ID).
		Int64("user_id", p.UserID).
		Str("title", p.Title).
		Str("url", p.URL).
		Str("website", p.Website).
		Str("description", p.Description).
		Bool("subscribed", p.Subscribed).
		Time("created_at", p.CreatedAt).
		Time("updated_at", p.UpdatedAt).
		Time("metadata_updated_at", p.MetaUpdatedAt.Time)
}

func (p *PodcastDB) toModel() *model.Podcast {
	return &model.Podcast{
		ID:          p.ID,
		Title:       p.Title,
		URL:         p.URL,
		Description: p.Description,
		Website:     p.Website,
		UpdatedAt:   p.UpdatedAt,
		Subscribed:  p.Subscribed,
		User:        model.User{ID: p.UserID},
	}
}

func podcastsFromDb(podcasts []PodcastDB) []model.Podcast {
	res := make([]model.Podcast, len(podcasts))
	for i, r := range podcasts {
		res[i] = *r.toModel()
	}

	return res
}

//------------------------------------------------------------------------------

// type PodcastMetaUpdateDB struct {
// 	Title       string `db:"title"`
// 	URL         string `db:"url"`
// 	Description string `db:"description"`
// 	Website     string `db:"website"`

// 	MetaUpdatedAt time.Time `db:"metadata_updated_at"`
// }

//------------------------------------------------------------------------------

// type PodcastsDB []PodcastDB

// func (s PodcastsDB) FindSubscribedPodcastByURL(url string) (PodcastDB, bool) {
// 	for _, sp := range s {
// 		if sp.URL == url && sp.Subscribed {
// 			return sp, true
// 		}
// 	}

// 	return PodcastDB{}, false
// }

// func (s PodcastsDB) FindPodcastByURL(url string) (PodcastDB, bool) {
// 	for _, sp := range s {
// 		if sp.URL == url {
// 			return sp, true
// 		}
// 	}

// 	return PodcastDB{}, false
// }

// func (s PodcastsDB) ToURLs() []string {
// 	res := make([]string, 0, len(s))
// 	for _, p := range s {
// 		res = append(res, p.URL)
// 	}

// 	return res
// }

// func (s PodcastsDB) ToMap() map[string]PodcastDB {
// 	res := make(map[string]PodcastDB)

// 	for _, p := range s {
// 		res[p.URL] = p
// 	}

// 	return res
// }

// func (s PodcastsDB) ToIDsMap() map[string]int64 {
// 	res := make(map[string]int64)

// 	for _, p := range s {
// 		res[p.URL] = p.ID
// 	}

// 	return res
// }

//------------------------------------------------------------------------------

type EpisodeDB struct {
	ID        int64          `db:"id"`
	PodcastID int64          `db:"podcast_id"`
	DeviceID  sql.NullInt64  `db:"device_id"`
	Title     string         `db:"title"`
	URL       string         `db:"url"`
	Action    string         `db:"action"`
	Started   sql.NullInt32  `db:"started"`
	Position  sql.NullInt32  `db:"position"`
	Total     sql.NullInt32  `db:"total"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
	GUID      sql.NullString `db:"guid"`

	PodcastURL   string         `db:"podcast_url"`
	PodcastTitle string         `db:"podcast_title"`
	DeviceName   sql.NullString `db:"device_name"`
}

func (e *EpisodeDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", e.ID).
		Int64("podcast_id", e.PodcastID).
		Any("device_id", e.DeviceID).
		Str("title", e.Title).
		Str("url", e.URL).
		Str("action", e.Action).
		Any("guid", e.GUID).
		Any("started", e.Started).
		Any("position", e.Position).
		Any("total", e.Total).
		Time("created_at", e.CreatedAt).
		Time("updated_at", e.UpdatedAt).
		Dict("podcast", zerolog.Dict().
			Str("podcast_url", e.PodcastURL).
			Str("podcast_title", e.PodcastTitle)).
		Any("device", e.DeviceName)
}

func (e *EpisodeDB) toModel() *model.Episode {
	var device *model.Device
	if e.DeviceID.Valid {
		device = &model.Device{
			ID:   e.DeviceID.Int64,
			Name: e.DeviceName.String,
		}
	}

	episode := &model.Episode{
		ID: e.ID,
		Podcast: &model.Podcast{
			ID:  e.PodcastID,
			URL: e.PodcastURL,
		},
		Device:    device,
		URL:       e.URL,
		Action:    e.Action,
		Timestamp: e.UpdatedAt,
		Started:   nil,
		Position:  nil,
		Total:     nil,
	}

	if e.GUID.Valid {
		episode.GUID = &e.GUID.String
	}

	if e.Action == "play" {
		episode.Started = &e.Started.Int32
		episode.Position = &e.Position.Int32
		episode.Total = &e.Total.Int32
	}

	return episode
}

//------------------------------------------------------------------------------

func episodesFromDb(episodes []EpisodeDB) []model.Episode {
	res := make([]model.Episode, len(episodes))
	for i, r := range episodes {
		res[i] = *r.toModel()
	}

	return res
}

//------------------------------------------------------------------------------

type UserDB struct {
	ID        int64     `db:"id"`
	UserName  string    `db:"username"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (u *UserDB) MarshalZerologObject(event *zerolog.Event) {
	pass := ""
	if u.Password != "" {
		pass = "***"
	}

	event.Int64("id", u.ID).
		Str("user_name", u.UserName).
		Str("Password", pass).
		Str("email", u.Email).
		Str("name", u.Name).
		Time("created_at", u.CreatedAt).
		Time("updated_at", u.UpdatedAt)
}

func (u *UserDB) toModel() *model.User {
	return &model.User{
		ID:       u.ID,
		UserName: u.UserName,
		Password: u.Password,
		Email:    u.Email,
		Name:     u.Name,
		Locked:   u.Password == model.UserLockedPassword,
	}
}

//------------------------------------------------------------------------------

func usersFromDb(users []UserDB) []model.User {
	res := make([]model.User, len(users))
	for i, r := range users {
		res[i] = *r.toModel()
	}

	return res
}

// ------------------------------------------------------------------------------

type SettingsDB struct {
	UserID    int64         `db:"user_id"`
	PodcastID sql.NullInt64 `db:"podcast_id"`
	EpisodeID sql.NullInt64 `db:"episode_id"`
	DeviceID  sql.NullInt64 `db:"device_id"`
	Scope     string        `db:"scope"`
	Key       string        `db:"key"`
	Value     string        `db:"value"`
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

//------------------------------------------------------------------------------
