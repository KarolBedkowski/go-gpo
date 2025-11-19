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

// ------------------------------------------------------

var CtxDBContextKey = any("CtxDBContextKey")

func WithCtx(ctx context.Context, dbctx DBContext) context.Context {
	db, ok := ctx.Value(CtxDBContextKey).(DBContext)
	if ok && db != nil {
		return ctx
	}

	return context.WithValue(ctx, CtxDBContextKey, dbctx)
}

func Ctx(ctx context.Context) (DBContext, bool) {
	value, ok := ctx.Value(CtxDBContextKey).(DBContext)
	if !ok || value == nil {
		return nil, false
	}

	return value, true
}

func MustCtx(ctx context.Context) DBContext {
	value, ok := ctx.Value(CtxDBContextKey).(DBContext)
	if !ok || value == nil {
		panic("no dbcontext in context")
	}

	return value
}

// ------------------------------------------------------

type DevicesRepository interface {
	GetDevice(ctx context.Context, userid int64, devicename string) (DeviceDB, error)
	SaveDevice(ctx context.Context, device *DeviceDB) (int64, error)
	ListDevices(ctx context.Context, userid int64) (DevicesDB, error)
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
	GetEpisode(ctx context.Context, userid, podcastid int64, episode string) (EpisodeDB, error)
	ListEpisodeActions(
		ctx context.Context, userid int64, deviceid, podcastid *int64, since time.Time, aggregated bool,
		lastelements uint,
	) ([]EpisodeDB, error)
	SaveEpisode(ctx context.Context, userid int64, episode ...EpisodeDB) error
	ListFavorites(ctx context.Context, userid int64) ([]EpisodeDB, error)
	GetLastEpisodeAction(ctx context.Context,
		userid, podcastid int64, excludeDelete bool) (EpisodeDB, error)
}

type PodcastsRepository interface {
	ListSubscribedPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	ListPodcasts(ctx context.Context, userid int64, since time.Time) (PodcastsDB, error)
	GetPodcast(ctx context.Context, userid int64, podcasturl string) (PodcastDB, error)
	SavePodcast(ctx context.Context, podcast *PodcastDB) (int64, error)
}

type SettingsRepository interface {
	ListSettings(ctx context.Context, userid int64, podcastid, episodeid, deviceid *int64, scope string,
	) ([]SettingsDB, error)
	// save (insert or update) or delete settings
	SaveSettings(ctx context.Context, sett *SettingsDB) error
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
