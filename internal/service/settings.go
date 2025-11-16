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
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
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

func (s SettingsSrv) GetSettings(ctx context.Context, key *model.SettingsKey) (model.Settings, error) {
	// validate
	if err := key.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate settings key to load failed").WithMeta("key", key)
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.db, func(dbctx repository.DBContext) (model.Settings, error) {
		setkey, err := s.newSettingKeys(ctx, dbctx, key)
		if err != nil {
			return nil, err
		}

		sett, err := s.settRepo.ListSettings(ctx, dbctx, setkey.userid, setkey.podcastid, setkey.episodeid,
			setkey.deviceid, key.Scope)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "failed get list settings")
		}

		settings := make(map[string]string)
		for _, s := range sett {
			settings[s.Key] = s.Value
		}

		return settings, nil
	})
}

// SaveSettings for `key` and values in `set`. If value is set to "" for given key - delete it.
func (s SettingsSrv) SaveSettings(ctx context.Context, cmd *command.ChangeSettingsCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate settings key to save failed")
	}

	key := model.NewSettingsKey(cmd.UserName, cmd.Scope, cmd.DeviceName, cmd.Podcast, cmd.Episode)
	settings := cmd.CombinedSetting()

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.db, func(dbctx repository.DBContext) error {
		setkey, err := s.newSettingKeys(ctx, dbctx, &key)
		if err != nil {
			return err
		}

		dbsett := repository.SettingsDB{
			UserID:    setkey.userid,
			PodcastID: setkey.podcastid,
			EpisodeID: setkey.episodeid,
			DeviceID:  setkey.deviceid,
			Scope:     key.Scope,
		}

		for key, value := range settings {
			dbsett.Key = key
			dbsett.Value = value

			if err := s.settRepo.SaveSettings(ctx, dbctx, &dbsett); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

//------------------------------------------------------------------------------

type settingsKeys struct {
	userid    int64
	podcastid *int64
	deviceid  *int64
	episodeid *int64
}

func (s SettingsSrv) newSettingKeys( //nolint:cyclop
	ctx context.Context,
	dbctx repository.DBContext,
	key *model.SettingsKey,
) (settingsKeys, error) {
	settkey := settingsKeys{}

	user, err := s.usersRepo.GetUser(ctx, dbctx, key.UserName)
	if errors.Is(err, repository.ErrNoData) {
		return settkey, ErrUnknownUser
	} else if err != nil {
		return settkey, aerr.ApplyFor(ErrRepositoryError, err)
	}

	settkey.userid = user.ID

	switch key.Scope {
	case "device":
		device, err := s.devicesRepo.GetDevice(ctx, dbctx, user.ID, key.DeviceName)
		if errors.Is(err, repository.ErrNoData) {
			return settkey, ErrUnknownDevice
		} else if err != nil {
			return settkey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		settkey.deviceid = &device.ID

	case "podcast":
		p, err := s.podcastsRepo.GetPodcast(ctx, dbctx, user.ID, key.Podcast)
		if errors.Is(err, repository.ErrNoData) {
			return settkey, ErrUnknownPodcast
		} else if err != nil {
			return settkey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		settkey.podcastid = &p.ID

	case "episode":
		p, err := s.podcastsRepo.GetPodcast(ctx, dbctx, user.ID, key.Podcast)
		if errors.Is(err, repository.ErrNoData) {
			return settkey, ErrUnknownEpisode
		} else if err != nil {
			return settkey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		settkey.podcastid = &p.ID

		e, err := s.episodesRepo.GetEpisode(ctx, dbctx, user.ID, p.ID, key.Episode)
		if errors.Is(err, repository.ErrNoData) {
			return settkey, ErrUnknownPodcast // TODO: fixme
		} else if err != nil {
			return settkey, aerr.ApplyFor(ErrRepositoryError, err)
		}

		settkey.episodeid = &e.ID
	case "account":
		// no extra data
	default:
		return settkey, aerr.NewSimple("unknown scope")
	}

	return settkey, nil
}
