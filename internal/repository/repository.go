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

	"gitlab.com/kabes/go-gpo/internal/model"
)

// ------------------------------------------------------

type DevicesRepository interface {
	GetDevice(ctx context.Context, userid int64, devicename string) (*model.Device, error)
	SaveDevice(ctx context.Context, device *model.Device) (int64, error)
	ListDevices(ctx context.Context, userid int64) ([]*model.Device, error)
	DeleteDevice(ctx context.Context, deviceid int64) error
	MarkSeen(ctx context.Context, ts time.Time, deviceid ...int64) error
}

type UsersRepository interface {
	GetUser(ctx context.Context, username string) (UserDB, error)
	SaveUser(ctx context.Context, user *UserDB) (int64, error)
	ListUsers(ctx context.Context, activeOnly bool) ([]UserDB, error)
	DeleteUser(ctx context.Context, userid int64) error
}

type EpisodesRepository interface {
	// GetEpisode from repository. episode can be episode url or guid.
	GetEpisode(ctx context.Context, userid, podcastid int64, episode string) (*model.Episode, error)
	ListEpisodeActions(
		ctx context.Context, userid int64, deviceid, podcastid *int64, since time.Time, aggregated bool,
		lastelements uint,
	) ([]model.Episode, error)
	SaveEpisode(ctx context.Context, userid int64, episode ...model.Episode) error
	ListFavorites(ctx context.Context, userid int64) ([]model.Episode, error)
	GetLastEpisodeAction(ctx context.Context,
		userid, podcastid int64, excludeDelete bool) (*model.Episode, error)
}

type PodcastsRepository interface {
	ListSubscribedPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	ListPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	GetPodcast(ctx context.Context, userid int64, podcasturl string) (PodcastDB, error)
	GetPodcastByID(ctx context.Context, userid, podcastid int64) (PodcastDB, error)
	SavePodcast(ctx context.Context, podcast *PodcastDB) (int64, error)
	// ListPodcastsToUpdate return list of url-s podcasts that need update (load title etc).
	ListPodcastsToUpdate(ctx context.Context, since time.Time) ([]string, error)
	UpdatePodcastsInfo(ctx context.Context, podcast *PodcastMetaUpdateDB) error
}

type SettingsRepository interface {
	ListSettings(ctx context.Context, userid int64, podcastid, episodeid, deviceid *int64, scope string,
	) (model.Settings, error)
	// save (insert or update) or delete settings
	SaveSettings(ctx context.Context,
		userid int64, podcastid, episodeid, deviceid *int64, scope, key, value string,
	) error
}

type SessionRepository interface {
	DeleteSession(ctx context.Context, sid string) error
	SaveSession(ctx context.Context, sid string, data []byte) error
	RegenerateSession(ctx context.Context, oldsid, newsid string) error
	CountSessions(ctx context.Context) (int, error)
	CleanSessions(ctx context.Context, maxLifeTime, maxLifeTimeForEmpty time.Duration) error
	ReadOrCreate(ctx context.Context, sid string) (session SessionDB, err error)
	SessionExists(ctx context.Context, sid string) (bool, error)
}

type Repository interface {
	DevicesRepository
	UsersRepository
	EpisodesRepository
	PodcastsRepository
	SettingsRepository
	SessionRepository
}

type MaintenanceRepository interface {
	Maintenance(ctx context.Context) error
}
