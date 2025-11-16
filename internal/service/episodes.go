//
// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type EpisodesSrv struct {
	db           *db.Database
	episodesRepo repository.EpisodesRepository
	devicesRepo  repository.DevicesRepository
	podcastsRepo repository.PodcastsRepository
	usersRepo    repository.UsersRepository
}

func NewEpisodesSrv(i do.Injector) (*EpisodesSrv, error) {
	return &EpisodesSrv{
		db:           do.MustInvoke[*db.Database](i),
		episodesRepo: do.MustInvoke[repository.EpisodesRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
	}, nil
}

// GetEpisodes return list of episodes for `username`, `podcast` and `devicename` (ignored).
// Return last action.
func (e *EpisodesSrv) GetEpisodes(ctx context.Context, username, devicename, podcast string,
) ([]model.Episode, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.Episode, error) {
		actions, err := e.getEpisodesActions(ctx, conn, username, devicename, podcast, time.Time{}, true, 0)
		if err != nil {
			return nil, err
		}

		episodes := make([]model.Episode, len(actions))
		for i, episode := range actions {
			episodes[i] = model.NewEpisodeFromDBModel(&episode)
		}

		return episodes, nil
	})
}

// AddAction save new actions.
// Podcasts and devices are cached and - if not exists for requested action - created.
func (e *EpisodesSrv) AddAction(ctx context.Context, username string, action ...model.Episode) error { //nolint:cyclop
	if username == "" {
		return ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, e.db, func(dbctx repository.DBContext) error {
		user, err := e.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		// cache devices and podcasts
		podcastscache, err := e.createPodcastsCache(ctx, dbctx, user.ID)
		if err != nil {
			return err
		}

		devicescache, err := e.createDevicesCache(ctx, dbctx, user.ID)
		if err != nil {
			return err
		}

		episodes := make([]repository.EpisodeDB, len(action))
		for idx, act := range action {
			episode := act.ToDBModel()

			episode.PodcastID, err = podcastscache.GetOrCreate(act.Podcast)
			if err != nil {
				return err
			}

			// devicecache handle nil for empty Device
			episode.DeviceID, err = devicescache.GetOrCreate(act.Device)
			if err != nil {
				return err
			}

			episodes[idx] = episode
		}

		if err = e.episodesRepo.SaveEpisode(ctx, dbctx, user.ID, episodes...); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		for _, deviceid := range devicescache.GetUsedValues() {
			if err := e.devicesRepo.MarkSeen(ctx, dbctx, time.Now().UTC(), *deviceid); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

// GetActions return list of episode actions for username and optional podcast and devicename.
// Device name is ignored as all devices are synced and should have the same data.
// Used by /api/2/episodes.
func (e *EpisodesSrv) GetActions(ctx context.Context, username, podcast, devicename string,
	since time.Time, aggregated bool,
) ([]model.Episode, error) {
	log.Ctx(ctx).Debug().Str("username", username).Str("devicename", devicename).
		Msgf("get actions since=%s aggregated=%v", since, aggregated)

	if username == "" {
		return nil, ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.Episode, error) {
		actions, err := e.getEpisodesActions(ctx, conn, username, devicename, podcast, since, aggregated, 0)
		if err != nil {
			return nil, err
		}

		res := make([]model.Episode, len(actions))
		for i, action := range actions {
			res[i] = model.NewEpisodeFromDBModel(&action)
		}

		return res, nil
	})
}

// GetUpdates return list of EpisodeUpdate for `username` and optionally `devicename` and `since`.
// if `includeActions` add to each episode last action.
// devicename is ignored but checked
// Used by /api/2/updates.
func (e *EpisodesSrv) GetUpdates(ctx context.Context, username, devicename string, since time.Time,
	includeActions bool,
) ([]model.EpisodeUpdate, error) {
	log.Ctx(ctx).Debug().Str("username", username).Str("devicename", devicename).
		Msgf("get episodes updates since %s includeActions %v", since, includeActions)

	if username == "" {
		return nil, ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.EpisodeUpdate, error) {
		episodes, err := e.getEpisodesActions(ctx, conn, username, devicename, "", since, true, 0)
		if err != nil {
			return nil, err
		}

		createFunc := model.NewEpisodeUpdateFromDBModel
		if includeActions {
			createFunc = model.NewEpisodeUpdateWithEpisodeFromDBModel
		}

		res := make([]model.EpisodeUpdate, len(episodes))
		for i, episodedb := range episodes {
			res[i] = createFunc(&episodedb)
		}

		return res, nil
	})
}

// GetLastActions return last `limit` actions for `username`.
func (e *EpisodesSrv) GetLastActions(ctx context.Context, username string, since time.Time, limit int,
) ([]model.Episode, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	actions, err := db.InConnectionR(ctx, e.db, func(dbctx repository.DBContext) ([]repository.EpisodeDB, error) {
		return e.getEpisodesActions(ctx, dbctx, username, "", "", since, false, limit)
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	res := make([]model.Episode, len(actions))
	for i, e := range actions {
		res[i] = model.NewEpisodeFromDBModel(&e)
	}

	return res, nil
}

func (e *EpisodesSrv) GetFavorites(ctx context.Context, username string) ([]model.Favorite, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	//nolint: wrapcheck
	return db.InConnectionR(ctx, e.db, func(dbctx repository.DBContext) ([]model.Favorite, error) {
		user, err := e.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		episodes, err := e.episodesRepo.ListFavorites(ctx, dbctx, user.ID)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		res := make([]model.Favorite, len(episodes))
		for i, e := range episodes {
			res[i] = model.NewFavoriteFromDBModel(&e)
		}

		return res, nil
	})
}

// ------------------------------------------------------

func (e *EpisodesSrv) getEpisodesActions(
	ctx context.Context,
	dbctx repository.DBContext,
	username, devicename, podcast string,
	since time.Time,
	aggregated bool,
	limit int,
) ([]repository.EpisodeDB, error) {
	user, err := e.usersRepo.GetUser(ctx, dbctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	_, err = e.getDeviceID(ctx, dbctx, user.ID, devicename)
	if err != nil {
		return nil, err
	}

	podcastid, err := e.getPodcastID(ctx, dbctx, user.ID, podcast)
	if err != nil {
		return nil, err
	}

	episodes, err := e.episodesRepo.ListEpisodeActions(ctx, dbctx, user.ID, nil, podcastid, since, aggregated,
		limit)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return episodes, nil
}

func (e *EpisodesSrv) createPodcastsCache(ctx context.Context, dbctx repository.DBContext,
	userid int64,
) (DynamicCache[string, int64], error) {
	podcasts, err := e.podcastsRepo.ListSubscribedPodcasts(ctx, dbctx, userid, time.Time{})
	if err != nil {
		return DynamicCache[string, int64]{}, aerr.Wrapf(err, "load podcasts into cache failed")
	}

	podcastscache := DynamicCache[string, int64]{
		items: podcasts.ToIDsMap(),
		creator: func(key string) (int64, error) {
			id, err := e.podcastsRepo.SavePodcast(ctx, dbctx,
				&repository.PodcastDB{UserID: userid, URL: key, Subscribed: true})
			if err != nil {
				return 0, aerr.Wrapf(err, "create new podcast failed")
			}

			return id, nil
		},
	}

	return podcastscache, nil
}

func (e *EpisodesSrv) createDevicesCache(ctx context.Context, dbctx repository.DBContext,
	userid int64,
) (DynamicCache[string, *int64], error) {
	devices, err := e.devicesRepo.ListDevices(ctx, dbctx, userid)
	if err != nil {
		return DynamicCache[string, *int64]{}, aerr.Wrapf(err, "load devices into cache failed")
	}

	items := make(map[string]*int64, len(devices))
	for _, d := range devices {
		items[d.Name] = &d.ID
	}

	devicescache := DynamicCache[string, *int64]{
		items: items,
		creator: func(key string) (*int64, error) {
			if key == "" {
				return nil, nil //nolint:nilnil
			}

			did, err := e.devicesRepo.SaveDevice(ctx, dbctx,
				&repository.DeviceDB{UserID: userid, Name: key, DevType: "other"})
			if err != nil {
				return nil, aerr.Wrapf(err, "create new device failed")
			}

			return &did, nil
		},
	}

	return devicescache, nil
}

func (e *EpisodesSrv) getDeviceID(
	ctx context.Context,
	dbctx repository.DBContext,
	userid int64,
	devicename string,
) (*int64, error) {
	if devicename == "" {
		return nil, nil //nolint:nilnil
	}

	device, err := e.devicesRepo.GetDevice(ctx, dbctx, userid, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if err := e.devicesRepo.MarkSeen(ctx, dbctx, time.Now().UTC(), device.ID); err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &device.ID, nil
}

func (e *EpisodesSrv) getPodcastID(
	ctx context.Context,
	dbctx repository.DBContext,
	userid int64,
	podcast string,
) (*int64, error) {
	if podcast == "" {
		return nil, nil //nolint:nilnil
	}

	p, err := e.podcastsRepo.GetPodcast(ctx, dbctx, userid, podcast)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownPodcast
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &p.ID, nil
}
