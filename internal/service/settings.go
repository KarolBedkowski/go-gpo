//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"errors"

	//	"gitlab.com/kabes/go-gpo/internal/model"
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

type SettingsSrv struct {
	db           *db.Database
	settRepo     repository.SettingsRepository
	usersRepo    repository.UsersRepository
	episodesRepo repository.EpisodesRepository
	devicesRepo  repository.DevicesRepository
	podcastsRepo repository.PodcastsRepository
}

func NewSettingsSrv(i do.Injector) (*SettingsSrv, error) {
	return &SettingsSrv{
		db:           do.MustInvoke[*db.Database](i),
		settRepo:     do.MustInvoke[repository.SettingsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		episodesRepo: do.MustInvoke[repository.EpisodesRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
	}, nil
}

func (s SettingsSrv) GetSettings(ctx context.Context, query *query.SettingsQuery) (model.Settings, error) {
	log.Ctx(ctx).Debug().Object("query", query).Msg("get settings")

	// validate
	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.db, func(ctx context.Context) (model.Settings, error) {
		key, err := s.load(ctx, query.UserName, query.Scope, query.DeviceName, query.Podcast, query.Episode)
		if err != nil {
			return nil, err
		}

		settings, err := s.settRepo.GetSettings(ctx, &key)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "failed get list settings")
		}

		return settings, nil
	})
}

// SaveSettings for `key` and values in `set`. If value is set to "" for given key - delete it.
func (s SettingsSrv) SaveSettings(ctx context.Context, cmd *command.ChangeSettingsCmd) error {
	log.Ctx(ctx).Debug().Object("cmd", cmd).Msg("save settings")

	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate settings key to save failed")
	}

	settings := cmd.CombinedSetting()

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.db, func(ctx context.Context) error {
		key, err := s.load(ctx, cmd.UserName, cmd.Scope, cmd.DeviceName, cmd.Podcast, cmd.Episode)
		if err != nil {
			return err
		}

		for skey, value := range settings {
			key.Key = skey

			if err := s.settRepo.SaveSettings(ctx, &key, value); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

func (s SettingsSrv) load( //nolint:cyclop
	ctx context.Context,
	username, scope, devicename, podcast, episode string,
) (model.SettingsKey, error) {
	skey := model.SettingsKey{Scope: scope}

	user, err := s.usersRepo.GetUser(ctx, username)
	if errors.Is(err, common.ErrNoData) {
		return skey, common.ErrUnknownUser
	} else if err != nil {
		return skey, aerr.ApplyFor(ErrRepositoryError, err)
	}

	skey.UserID = user.ID

	switch scope {
	case "device":
		device, err := s.devicesRepo.GetDevice(ctx, user.ID, devicename)
		if errors.Is(err, common.ErrNoData) {
			return skey, common.ErrUnknownDevice
		} else if err != nil {
			return skey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		skey.DeviceID = &device.ID

	case "podcast":
		p, err := s.podcastsRepo.GetPodcast(ctx, user.ID, podcast)
		if errors.Is(err, common.ErrNoData) {
			return skey, common.ErrUnknownPodcast
		} else if err != nil {
			return skey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		skey.PodcastID = &p.ID

	case "episode":
		p, err := s.podcastsRepo.GetPodcast(ctx, user.ID, podcast)
		if errors.Is(err, common.ErrNoData) {
			return skey, common.ErrUnknownEpisode
		} else if err != nil {
			return skey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		skey.PodcastID = &p.ID

		e, err := s.episodesRepo.GetEpisode(ctx, user.ID, p.ID, episode)
		if errors.Is(err, common.ErrNoData) {
			return skey, common.ErrUnknownPodcast
		} else if err != nil {
			return skey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		skey.EpisodeID = &e.ID
	case "account":
		// no extra data
	default:
		return skey, aerr.New("unknown scope").WithTag(aerr.ValidationError)
	}

	return skey, nil
}

//------------------------------------------------------------------------------
