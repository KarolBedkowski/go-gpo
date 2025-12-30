package service

//
// maintenance.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type MaintenanceSrv struct {
	db           *db.Database
	maintRepo    repository.Maintenance
	usersRepo    repository.Users
	devicesRepo  repository.Devices
	podcastsRepo repository.Podcasts
	episodesRepo repository.Episodes
	settingsRepo repository.Settings
}

func NewMaintenanceSrv(i do.Injector) (*MaintenanceSrv, error) {
	return &MaintenanceSrv{
		db:           do.MustInvoke[*db.Database](i),
		maintRepo:    do.MustInvoke[repository.Maintenance](i),
		usersRepo:    do.MustInvoke[repository.Users](i),
		devicesRepo:  do.MustInvoke[repository.Devices](i),
		podcastsRepo: do.MustInvoke[repository.Podcasts](i),
		episodesRepo: do.MustInvoke[repository.Episodes](i),
		settingsRepo: do.MustInvoke[repository.Settings](i),
	}, nil
}

func (m *MaintenanceSrv) MaintainDatabase(ctx context.Context) error {
	_, err := db.InConnectionR(ctx, m.db, func(ctx context.Context) (any, error) {
		return nil, m.maintRepo.Maintenance(ctx)
	})
	if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	return nil
}

func (m *MaintenanceSrv) ExportAll(ctx context.Context) ([]model.ExportStruct, error) {
	res, err := db.InConnectionR(ctx, m.db, func(ctx context.Context) ([]model.ExportStruct, error) {
		res := []model.ExportStruct{}

		users, err := m.usersRepo.ListUsers(ctx, false)
		if err != nil {
			return nil, aerr.Wrapf(err, "get users list error")
		}

		if len(users) == 0 {
			return nil, aerr.New("no users")
		}

		for _, user := range users {
			esu := model.ExportStruct{User: user}

			esu.Devices, err = m.devicesRepo.ListDevices(ctx, user.ID)
			if err != nil {
				return nil, aerr.Wrapf(err, "get users devices error").WithMeta("user_id", user.ID)
			}

			esu.Podcasts, err = m.podcastsRepo.ListPodcasts(ctx, user.ID, time.Time{})
			if err != nil {
				return nil, aerr.Wrapf(err, "get user podcasts error").WithMeta("user_id", user.ID)
			}

			esu.Episodes, err = m.episodesRepo.ListEpisodeActions(ctx, user.ID, nil, nil, time.Time{}, false, 0)
			if err != nil {
				return nil, aerr.Wrapf(err, "get user episodes error").WithMeta("user_id", user.ID)
			}

			esu.Settings, err = m.settingsRepo.GetAllSettings(ctx, user.ID)
			if err != nil {
				return nil, aerr.Wrapf(err, "get user settings error").WithMeta("user_id", user.ID)
			}

			res = append(res, esu)
		}

		return res, nil
	})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return res, nil
}

func (m *MaintenanceSrv) ImportAll(ctx context.Context, data []model.ExportStruct) error {
	logger := zerolog.Ctx(ctx)

	err := db.InTransaction(ctx, m.db, func(ctx context.Context) error {
		for _, record := range data {
			logger.Info().Msgf("loading user %q", record.User.UserName)

			record.User.ID = 0

			uid, err := m.usersRepo.SaveUser(ctx, &record.User)
			if err != nil {
				return aerr.Wrapf(err, "save users error").WithMeta("username", record.User.Name)
			}

			logger.Debug().Msgf("loading user %q - devices...", record.User.UserName)

			devmap, err := m.importDevices(ctx, record.Devices, uid)
			if err != nil {
				return err
			}

			logger.Debug().Msgf("loading user %q - podcasts...", record.User.UserName)

			podcastsmap, err := m.importPodcasts(ctx, record.Podcasts, uid)
			if err != nil {
				return err
			}

			logger.Debug().Msgf("loading user %q - episodes...", record.User.UserName)

			ep := remapEpisodes(record.Episodes, podcastsmap, devmap)

			err = m.episodesRepo.SaveEpisode(ctx, uid, ep...)
			if err != nil {
				return aerr.Wrapf(err, "save episodes error")
			}

			err = m.importSettings(ctx, &record, uid, podcastsmap, devmap)
			if err != nil {
				return aerr.Wrapf(err, "import settings error")
			}
		}

		return nil
	})
	if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	return nil
}

func (m *MaintenanceSrv) importDevices(
	ctx context.Context,
	devices []model.Device,
	uid int64,
) (map[int64]int64, error) {
	devmap := make(map[int64]int64, len(devices))

	for _, d := range devices {
		oldid := d.ID
		d.ID = 0
		d.User.ID = uid

		did, err := m.devicesRepo.SaveDevice(ctx, &d)
		if err != nil {
			return nil, aerr.Wrapf(err, "save device error").WithMeta("devicename", d.Name)
		}

		devmap[oldid] = did
	}

	return devmap, nil
}

func (m *MaintenanceSrv) importPodcasts(
	ctx context.Context,
	podcasts model.Podcasts,
	uid int64,
) (map[int64]int64, error) {
	podcastsmap := make(map[int64]int64, len(podcasts))

	for _, p := range podcasts {
		oldid := p.ID
		p.ID = 0
		p.User.ID = uid

		pid, err := m.podcastsRepo.SavePodcast(ctx, &p)
		if err != nil {
			return nil, aerr.Wrapf(err, "save podcasts error").WithMeta("podcast_url", p.URL)
		}

		podcastsmap[oldid] = pid
	}

	return podcastsmap, nil
}

func (m *MaintenanceSrv) importSettings(
	ctx context.Context,
	data *model.ExportStruct,
	uid int64,
	podcastsmap, devmap map[int64]int64,
) error {
	for _, usett := range data.Settings {
		usett.UserID = uid

		if usett.PodcastID != nil {
			v := podcastsmap[*usett.PodcastID]
			usett.PodcastID = &v
		}

		if usett.DeviceID != nil {
			v := devmap[*usett.DeviceID]
			usett.DeviceID = &v
		}

		if usett.EpisodeID != nil {
			if usett.PodcastID == nil {
				return aerr.New("missing podcast for episode").WithMeta("episode_id", usett.EpisodeID)
			}

			e, ok := data.FindEpisode(*usett.EpisodeID)
			if !ok {
				return aerr.New("episode not found").WithMeta("episode_id", usett.EpisodeID)
			}

			uepisode, err := m.episodesRepo.GetEpisode(ctx, uid, *usett.PodcastID, e.URL)
			if err != nil {
				return aerr.Wrapf(err, "failed to find episode").WithMeta("userid", uid,
					"podcastid", *usett.PodcastID, "episode_url", e.URL)
			}

			usett.EpisodeID = &uepisode.ID
		}

		key := usett.ToKey()

		err := m.settingsRepo.SaveSettings(ctx, &key, usett.Value)
		if err != nil {
			return aerr.Wrapf(err, "save settings error").WithMeta("setting", usett)
		}
	}

	return nil
}

func remapEpisodes(episodes []model.Episode, podcastsmap, devmap map[int64]int64) []model.Episode {
	ep := make([]model.Episode, len(episodes))
	for i, e := range episodes {
		e.Podcast.ID = podcastsmap[e.Podcast.ID]
		if e.Device != nil {
			e.Device.ID = devmap[e.Device.ID]
		}

		ep[i] = e
	}

	return ep
}
