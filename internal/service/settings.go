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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Settings struct {
	db           *db.Database
	settRepo     repository.SettingsRepository
	usersRepo    repository.UsersRepository
	episodesRepo repository.EpisodesRepository
	devicesRepo  repository.DevicesRepository
	podcastsRepo repository.PodcastsRepository
}

func NewSettingsServiceI(i do.Injector) (*Settings, error) {
	return &Settings{
		db:           do.MustInvoke[*db.Database](i),
		settRepo:     do.MustInvoke[repository.SettingsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		episodesRepo: do.MustInvoke[repository.EpisodesRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
	}, nil
}

func (s Settings) GetSettings(ctx context.Context, key *model.SettingsKey) (map[string]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	setkey, err := s.newSettingKeys(ctx, conn, key)
	if err != nil {
		return nil, err
	}

	sett, err := s.settRepo.ListSettings(ctx, conn, setkey.userid, setkey.podcastid, setkey.episodeid,
		setkey.deviceid, key.Scope)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	settings := make(map[string]string)
	for _, s := range sett {
		settings[s.Key] = s.Value
	}

	return settings, nil
}

func (s Settings) SaveSettings(ctx context.Context, key *model.SettingsKey, set map[string]string, del []string) error {
	err := s.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		setkey, err := s.newSettingKeys(ctx, dbctx, key)
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

		for key, value := range set {
			dbsett.Key = key
			dbsett.Value = value

			if err := s.settRepo.SaveSettings(ctx, dbctx, &dbsett); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		dbsett.Value = ""

		for _, key := range del {
			dbsett.Key = key

			if err := s.settRepo.SaveSettings(ctx, dbctx, &dbsett); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})

	return err //nolint:wrapcheck
}

type settingsKeys struct {
	userid    int64
	podcastid *int64
	deviceid  *int64
	episodeid *int64
}

func (s Settings) newSettingKeys( //nolint:cyclop
	ctx context.Context,
	dbctx repository.DBContext,
	key *model.SettingsKey,
) (settingsKeys, error) {
	settkey := settingsKeys{}

	user, err := s.usersRepo.GetUser(ctx, dbctx, key.Username)
	if errors.Is(err, repository.ErrNoData) {
		return settkey, ErrUnknownUser
	} else if err != nil {
		return settkey, aerr.ApplyFor(ErrRepositoryError, err)
	}

	settkey.userid = user.ID

	switch key.Scope {
	case "device":
		device, err := s.devicesRepo.GetDevice(ctx, dbctx, user.ID, key.Device)
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
