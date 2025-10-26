package repository

//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type queryer interface {
	sqlx.QueryerContext
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

type DBContext interface {
	// sqlx.ExtContext
	sqlx.QueryerContext
	sqlx.ExecerContext

	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

type DevicesRepository interface {
	GetDevice(ctx context.Context, userid int64, devicename string) (DeviceDB, error)
	SaveDevice(ctx context.Context, device *DeviceDB) (int64, error)
	ListDevices(ctx context.Context, userid int64) (DevicesDB, error)
}

type UsersRepository interface {
	GetUser(ctx context.Context, username string) (UserDB, error)
	SaveUser(ctx context.Context, user *UserDB) (int64, error)
}

type EpisodesRepository interface {
	GetEpisodes(ctx context.Context, userid, deviceid, podcastid int64, since time.Time, aggregated bool,
	) ([]EpisodeDB, error)
	SaveEpisode(ctx context.Context, userid int64, episode ...EpisodeDB) error
}

type SubscribedRepository interface {
	GetSubscribedPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	GetPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	GetPodcast(ctx context.Context, userid int64, podcasturl string) (PodcastDB, error)
	SavePodcast(ctx context.Context, user, device string, podcast ...PodcastDB) error
}

type SettingsRepository interface {
	GetSettings(ctx context.Context, userid int64, scope, key string) (SettingsDB, error)
	SaveSettings(ctx context.Context, sett *SettingsDB) error
}

type Repository interface {
	DevicesRepository
	UsersRepository
	EpisodesRepository
	SubscribedRepository
	SettingsRepository
}
