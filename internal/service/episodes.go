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

type Episodes struct {
	db           *db.Database
	episodesRepo repository.EpisodesRepository
	devicesRepo  repository.DevicesRepository
	podcastsRepo repository.PodcastsRepository
	usersRepo    repository.UsersRepository
}

func NewEpisodesServiceI(i do.Injector) (*Episodes, error) {
	return &Episodes{
		db:           do.MustInvoke[*db.Database](i),
		episodesRepo: do.MustInvoke[repository.EpisodesRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
	}, nil
}

// GetPodcastEpisodes return list of episodes for `uername`, `podcast` and `devicename`. Return last action.
func (e *Episodes) GetPodcastEpisodes(ctx context.Context, username, podcast, devicename string,
) ([]model.Episode, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.Episode, error) {
		actions, err := e.getEpisodesActionsInternal(ctx, conn, username, podcast, devicename, time.Time{}, true, 0)
		if err != nil {
			return nil, err
		}

		if len(actions) == 0 {
			return []model.Episode{}, nil
		}

		episodes := make([]model.Episode, len(actions))
		for i, episode := range actions {
			episodes[i] = model.NewEpisodeFromDBModel(&episode)
		}

		return episodes, nil
	})
}

// SaveEpisodesActions save new actions.
// Podcasts and devices are cached and - if not exists for requested action - created.
func (e *Episodes) SaveEpisodesActions(ctx context.Context, username string, action ...model.Episode) error {
	//nolint:wrapcheck
	return e.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		user, err := e.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		// cache devices and podcasts
		podcasts, err := e.podcastsRepo.ListSubscribedPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return err
		}

		podcastscache := DynamicCache[string, int64]{
			items: podcasts.ToIDsMap(),
			creator: func(key string) (int64, error) {
				id, err := e.podcastsRepo.SavePodcast(ctx, dbctx,
					&repository.PodcastDB{UserID: user.ID, URL: key, Subscribed: true})
				if err != nil {
					return 0, aerr.Wrapf(err, "create new podcast failed")
				}

				return id, nil
			},
		}

		devices, err := e.devicesRepo.ListDevices(ctx, dbctx, user.ID)
		if err != nil {
			return err
		}

		devicescache := DynamicCache[string, int64]{
			items: devices.ToIDsMap(),
			creator: func(key string) (int64, error) {
				did, err := e.devicesRepo.SaveDevice(ctx, dbctx,
					&repository.DeviceDB{UserID: user.ID, Name: key, DevType: "other"})
				if err != nil {
					return 0, aerr.Wrapf(err, "create new device failed")
				}

				return did, nil
			},
		}

		episodes := make([]repository.EpisodeDB, len(action))
		for idx, act := range action {
			episode := act.ToDBModel()

			episode.PodcastID, err = podcastscache.GetOrCreate(act.Podcast)
			if err != nil {
				return err
			}

			episode.DeviceID, err = devicescache.GetOrCreate(act.Device)
			if err != nil {
				return err
			}

			episodes[idx] = episode
		}

		if err = e.episodesRepo.SaveEpisode(ctx, dbctx, user.ID, episodes...); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})
}

// GetEpisodesActions return list of episode actions for username and optional podcast and devicename.
// If devicename is not empty - get actions from other devices.
func (e *Episodes) GetEpisodesActions(ctx context.Context, username, podcast, devicename string,
	since time.Time, aggregated bool,
) ([]model.Episode, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.Episode, error) {
		actions, err := e.getEpisodesActionsInternal(ctx, conn, username, podcast, devicename, since, aggregated, 0)
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

// GetEpisodesUpdates return list of EpisodeUpdate for `username` and optionally `devicename` and `since`.
// if `includeActions` add to each episode last action.
func (e *Episodes) GetEpisodesUpdates(ctx context.Context, username, devicename string, since time.Time,
	includeActions bool,
) ([]model.EpisodeUpdate, error) {
	log.Ctx(ctx).Debug().Str("username", username).Str("devicename", devicename).
		Msgf("get episodes updates since %s includeActions %v", since, includeActions)

	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(conn repository.DBContext) ([]model.EpisodeUpdate, error) {
		episodes, err := e.getEpisodesActionsInternal(ctx, conn, username, "", devicename, since, true, 0)
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
func (e *Episodes) GetLastActions(ctx context.Context, username string, since time.Time, limit int,
) ([]model.Episode, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(dbctx repository.DBContext) ([]model.Episode, error) {
		episodes, err := e.getEpisodesActionsInternal(ctx, dbctx, username, "", "", since, false, limit)
		if err != nil {
			return nil, err
		}

		res := make([]model.Episode, len(episodes))
		for i, e := range episodes {
			res[i] = model.NewEpisodeFromDBModel(&e)
		}

		return res, nil
	})
}

func (e *Episodes) GetFavorites(ctx context.Context, username string) ([]model.Favorite, error) {
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

		res := make([]model.Favorite, 0, len(episodes))
		for i, e := range episodes {
			res[i] = model.NewFavoriteFromDBModel(&e)
		}

		return res, nil
	})
}

func (e *Episodes) getEpisodesActionsInternal(
	ctx context.Context,
	dbctx repository.DBContext,
	username, podcast, devicename string,
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

	var deviceid *int64

	if devicename != "" {
		device, err := e.devicesRepo.GetDevice(ctx, dbctx, user.ID, devicename)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownDevice
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		deviceid = &device.ID
	}

	var podcastid *int64

	if podcast != "" {
		p, err := e.podcastsRepo.GetPodcast(ctx, dbctx, user.ID, podcast)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownPodcast
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcastid = &p.ID
	}

	episodes, err := e.episodesRepo.ListEpisodeActions(ctx, dbctx, user.ID, deviceid, podcastid, since, aggregated,
		limit)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return episodes, nil
}
