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
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Name      string    `db:"name"`
	DevType   string    `db:"dev_type"`
	Caption   string    `db:"caption"`
	ID        int32     `db:"id"`
	UserID    int32     `db:"user_id"`

	Subscriptions int `db:"subscriptions"`

	User *UserDB `db:"user"`
}

func (d *DeviceDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("id", d.ID).
		Int32("user_id", d.UserID).
		Str("name", d.Name).
		Str("type", d.DevType).
		Str("caption", d.Caption).
		Time("created_at", d.CreatedAt).
		Time("updated_at", d.UpdatedAt).
		Int("subscriptions", d.Subscriptions)

	if d.User != nil {
		event.Object("user", d.User)
	}
}

func (d *DeviceDB) toModel() *model.Device {
	var user *model.User
	if d.User != nil {
		user = d.User.toModel()
	}

	return &model.Device{
		ID:            d.ID,
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
		UpdatedAt:     d.UpdatedAt,
		User:          user,
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
	CreatedAt     time.Time    `db:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at"`
	MetaUpdatedAt sql.NullTime `db:"metadata_updated_at"`
	Title         string       `db:"title"`
	URL           string       `db:"url"`
	Description   string       `db:"description"`
	Website       string       `db:"website"`

	ID     int32 `db:"id"`
	UserID int32 `db:"user_id"`

	Subscribed bool `db:"subscribed"`
}

func (p *PodcastDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("id", p.ID).
		Int32("user_id", p.UserID).
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

type EpisodeDB struct {
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Title     string    `db:"title"`
	URL       string    `db:"url"`
	Action    string    `db:"action"`

	PodcastURL   string         `db:"podcast_url"`
	PodcastTitle string         `db:"podcast_title"`
	GUID         sql.NullString `db:"guid"`

	DeviceName sql.NullString `db:"device_name"`
	DeviceID   sql.NullInt32  `db:"device_id"`
	ID         int32          `db:"id"`
	PodcastID  int32          `db:"podcast_id"`
	Started    sql.NullInt32  `db:"started"`
	Position   sql.NullInt32  `db:"position"`
	Total      sql.NullInt32  `db:"total"`
}

func (e *EpisodeDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("id", e.ID).
		Int32("podcast_id", e.PodcastID).
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
			ID:   e.DeviceID.Int32,
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
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	UserName  string    `db:"username"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	ID        int32     `db:"id"`
}

func (u *UserDB) MarshalZerologObject(event *zerolog.Event) {
	pass := ""
	if u.Password != "" {
		pass = "***"
	}

	event.Int32("id", u.ID).
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
	Scope     string        `db:"scope"`
	Key       string        `db:"key"`
	Value     string        `db:"value"`
	PodcastID sql.NullInt32 `db:"podcast_id"`
	EpisodeID sql.NullInt32 `db:"episode_id"`
	DeviceID  sql.NullInt32 `db:"device_id"`
	UserID    int32         `db:"user_id"`
}

func (s SettingsDB) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("user_id", s.UserID).
		Any("podcast_id", s.PodcastID).
		Any("episode_id", s.EpisodeID).
		Any("device_id", s.DeviceID).
		Str("scope", s.Scope).
		Str("key", s.Key).
		Str("value", s.Value)
}

//------------------------------------------------------------------------------
