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

type Queryer interface {
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
	GetDevice(ctx context.Context, db DBContext, userid int64, devicename string) (DeviceDB, error)
	SaveDevice(ctx context.Context, db DBContext, device *DeviceDB) (int64, error)
	ListDevices(ctx context.Context, db DBContext, userid int64) (DevicesDB, error)
}

type UsersRepository interface {
	GetUser(ctx context.Context, db DBContext, username string) (UserDB, error)
	SaveUser(ctx context.Context, db DBContext, user *UserDB) (int64, error)
	ListUsers(ctx context.Context, db DBContext, activeOnly bool) ([]UserDB, error)
}

type EpisodesRepository interface {
	// GetEpisode from repository. episode can be episode url or guid.
	GetEpisode(ctx context.Context, dbctx DBContext, userid, podcastid int64, episode string) (EpisodeDB, error)
	ListEpisodeActions(
		ctx context.Context, db DBContext, userid int64, deviceid, podcastid *int64, since time.Time, aggregated bool,
		lastelements int,
	) ([]EpisodeDB, error)
	SaveEpisode(ctx context.Context, db DBContext, userid int64, episode ...EpisodeDB) error
	ListFavorites(ctx context.Context, db DBContext, userid int64) ([]EpisodeDB, error)
}

type PodcastsRepository interface {
	ListSubscribedPodcasts(ctx context.Context, db DBContext, userid int64, since time.Time) (PodcastsDB, error)
	ListPodcasts(ctx context.Context, db DBContext, userid int64, since time.Time) (PodcastsDB, error)
	GetPodcast(ctx context.Context, db DBContext, userid int64, podcasturl string) (PodcastDB, error)
	SavePodcast(ctx context.Context, db DBContext, podcast *PodcastDB) (int64, error)
}

type SettingsRepository interface {
	ListSettings(ctx context.Context, db DBContext, userid int64, podcastid, episodeid, deviceid *int64, scope string,
	) ([]SettingsDB, error)
	// save (insert or update) or delete settings
	SaveSettings(ctx context.Context, db DBContext, sett *SettingsDB) error
}

type SessionRepository interface {
	DeleteSession(ctx context.Context, db DBContext, sid string) error
	SaveSession(ctx context.Context, db DBContext, sid string, data []byte) error
	RegenerateSession(ctx context.Context, db DBContext, oldsid, newsid string) error
	CountSessions(ctx context.Context, db DBContext) (int, error)
	CleanSessions(ctx context.Context, db DBContext, maxLifeTime, maxLifeTimeForEmpty time.Duration) error
	ReadOrCreate(ctx context.Context, db DBContext, sid string) (data []byte, createAt time.Time, err error)
	SessionExists(ctx context.Context, db DBContext, sid string) (bool, error)
}

type Repository interface {
	DevicesRepository
	UsersRepository
	EpisodesRepository
	PodcastsRepository
	SettingsRepository
	SessionRepository
}
