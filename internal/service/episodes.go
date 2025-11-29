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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
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
func (e *EpisodesSrv) GetEpisodes(ctx context.Context, query *query.GetEpisodesQuery) ([]model.Episode, error) {
	log.Ctx(ctx).Debug().Object("query", query).Msg("get episodes")

	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(ctx context.Context) ([]model.Episode, error) {
		return e.getEpisodes(
			ctx,
			query.UserName,
			query.DeviceName,
			query.Podcast,
			query.Since,
			query.Aggregated,
			query.Limit,
		)
	})
}

func (e *EpisodesSrv) GetEpisodesByPodcast(ctx context.Context, query *query.GetEpisodesByPodcastQuery,
) ([]model.Episode, error) {
	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}
	//nolint:wrapcheck
	return db.InConnectionR(ctx, e.db, func(ctx context.Context) ([]model.Episode, error) {
		user, err := e.usersRepo.GetUser(ctx, query.UserName)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		episodes, err := e.episodesRepo.ListEpisodeActions(ctx, user.ID, nil,
			&query.PodcastID, query.Since, query.Aggregated, query.Limit)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return episodes, nil
	})
}

// AddAction save new actions.
// Podcasts and devices are cached and - if not exists for requested action - created.
func (e *EpisodesSrv) AddAction(ctx context.Context, cmd *command.AddActionCmd) error { //nolint:cyclop
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("username", cmd.UserName).Msgf("add actions: %d", len(cmd.Actions))

	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate command failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, e.db, func(ctx context.Context) error {
		user, err := e.usersRepo.GetUser(ctx, cmd.UserName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		// cache devices and podcasts
		podcastscache, err := e.createPodcastsCache(ctx, user)
		if err != nil {
			return err
		}

		devicescache, err := e.createDevicesCache(ctx, user)
		if err != nil {
			return err
		}

		episodes := make([]model.Episode, len(cmd.Actions))
		for idx, act := range cmd.Actions {
			episode := act

			episode.Podcast.ID, err = podcastscache.GetOrCreate(act.Podcast.URL)
			if err != nil {
				return err
			}

			// devicecache handle nil for empty Device
			if act.Device != nil {
				did, err := devicescache.GetOrCreate(act.Device.Name)
				if err != nil {
					return err
				}

				episode.Device.ID = *did
			}

			episodes[idx] = episode
		}

		if err = e.episodesRepo.SaveEpisode(ctx, user.ID, episodes...); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		for _, deviceid := range devicescache.GetUsedValues() {
			if err := e.devicesRepo.MarkSeen(ctx, time.Now().UTC(), *deviceid); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

// GetUpdates return list of EpisodeUpdate for `username` and optionally `devicename` and `since`.
// if `includeActions` add to each episode last action.
// devicename is ignored but checked
// Used by /api/2/updates.
func (e *EpisodesSrv) GetUpdates(ctx context.Context, query *query.GetEpisodeUpdatesQuery,
) ([]model.EpisodeUpdate, error) {
	log.Ctx(ctx).Debug().Object("query", query).Msg("get episodes updates")

	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}

	episodes, err := db.InConnectionR(ctx, e.db, func(ctx context.Context) ([]model.Episode, error) {
		return e.getEpisodes(ctx, query.UserName, query.DeviceName, "", query.Since, true, 0)
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	createFunc := model.NewEpisodeUpdate
	if query.IncludeActions {
		createFunc = model.NewEpisodeUpdateWithEpisode
	}

	return common.Map(episodes, createFunc), nil
}

// GetLastActions return last `limit` actions for `username`.
func (e *EpisodesSrv) GetLastActions(ctx context.Context, query *query.GetLastEpisodesActionsQuery,
) ([]model.EpisodeLastAction, error) {
	log.Ctx(ctx).Debug().Object("query", query).Msg("get episodes")

	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}

	episodes, err := db.InConnectionR(ctx, e.db, func(ctx context.Context) ([]model.Episode, error) {
		return e.getEpisodes(ctx, query.UserName, "", "", query.Since, true, query.Limit)
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return common.Map(episodes, model.NewEpisodeLastAction), nil
}

func (e *EpisodesSrv) GetFavorites(ctx context.Context, username string) ([]model.Favorite, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	episodes, err := db.InConnectionR(ctx, e.db, func(ctx context.Context) ([]model.Episode, error) {
		user, err := e.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		episodes, err := e.episodesRepo.ListFavorites(ctx, user.ID)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return episodes, nil
	})
	if err != nil {
		return nil, err //nolint: wrapcheck
	}

	return common.Map(episodes, model.NewFavoriteFromModel), nil
}

// ------------------------------------------------------

func (e *EpisodesSrv) getEpisodes(
	ctx context.Context,
	username, devicename, podcast string,
	since time.Time,
	aggregated bool,
	limit uint,
) ([]model.Episode, error) {
	user, err := e.usersRepo.GetUser(ctx, username)
	if errors.Is(err, common.ErrNoData) {
		return nil, common.ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	// check device
	_, err = e.getDeviceID(ctx, user.ID, devicename)
	if err != nil {
		return nil, err
	}

	podcastid, err := e.getPodcastID(ctx, user.ID, podcast)
	if err != nil {
		return nil, err
	}

	episodes, err := e.episodesRepo.ListEpisodeActions(ctx, user.ID, nil, podcastid, since, aggregated,
		limit)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return episodes, nil
}

func (e *EpisodesSrv) createPodcastsCache(ctx context.Context, user *model.User) (DynamicCache[string, int64], error) {
	podcasts, err := e.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return DynamicCache[string, int64]{}, aerr.Wrapf(err, "load podcasts into cache failed").
			WithMeta("user_id", user.ID)
	}

	podcastscache := DynamicCache[string, int64]{
		items: podcasts.ToIDsMap(),
		creator: func(key string) (int64, error) {
			id, err := e.podcastsRepo.SavePodcast(ctx,
				&model.Podcast{User: *user, URL: key, Subscribed: true})
			if err != nil {
				return 0, aerr.Wrapf(err, "create new podcast failed").WithMeta("podcast_url", key, "user_id", user.ID)
			}

			return id, nil
		},
	}

	return podcastscache, nil
}

func (e *EpisodesSrv) createDevicesCache(ctx context.Context, user *model.User) (DynamicCache[string, *int64], error) {
	devices, err := e.devicesRepo.ListDevices(ctx, user.ID)
	if err != nil {
		return DynamicCache[string, *int64]{}, aerr.Wrapf(err, "load devices into cache failed").
			WithMeta("user_id", user.ID)
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

			did, err := e.devicesRepo.SaveDevice(ctx,
				&model.Device{User: user, Name: key, DevType: "other"})
			if err != nil {
				return nil, aerr.Wrapf(err, "create new device failed").WithMeta("device_name", key, "user_id", user.ID)
			}

			return &did, nil
		},
	}

	return devicescache, nil
}

func (e *EpisodesSrv) getDeviceID(
	ctx context.Context,
	userid int64,
	devicename string,
) (*int64, error) {
	if devicename == "" {
		return nil, nil //nolint:nilnil
	}

	device, err := e.devicesRepo.GetDevice(ctx, userid, devicename)
	if errors.Is(err, common.ErrNoData) {
		return nil, common.ErrUnknownDevice
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if err := e.devicesRepo.MarkSeen(ctx, time.Now().UTC(), device.ID); err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &device.ID, nil
}

func (e *EpisodesSrv) getPodcastID(
	ctx context.Context,
	userid int64,
	podcast string,
) (*int64, error) {
	if podcast == "" {
		return nil, nil //nolint:nilnil
	}

	p, err := e.podcastsRepo.GetPodcast(ctx, userid, podcast)
	if errors.Is(err, common.ErrNoData) {
		return nil, common.ErrUnknownPodcast
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &p.ID, nil
}
