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
	"gitlab.com/kabes/go-gpo/internal"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
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

	setkey := newSettingsKeysFromQuery(query)

	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.db, func(ctx context.Context) (model.Settings, error) {
		if err := s.load(ctx, &setkey); err != nil {
			return nil, err
		}

		sett, err := s.settRepo.ListSettings(ctx, setkey.userid, setkey.podcastid, setkey.episodeid,
			setkey.deviceid, query.Scope)
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
	log.Ctx(ctx).Debug().Object("cmd", cmd).Msg("save settings")

	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate settings key to save failed")
	}

	setkey := newSettingsKeysFromCmd(cmd)
	settings := cmd.CombinedSetting()

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.db, func(ctx context.Context) error {
		if err := s.load(ctx, &setkey); err != nil {
			return err
		}

		dbsett := repository.SettingsDB{
			UserID:    setkey.userid,
			PodcastID: setkey.podcastid,
			EpisodeID: setkey.episodeid,
			DeviceID:  setkey.deviceid,
			Scope:     setkey.scope,
		}

		for key, value := range settings {
			dbsett.Key = key
			dbsett.Value = value

			if err := s.settRepo.SaveSettings(ctx, &dbsett); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

func (s SettingsSrv) load( //nolint:cyclop
	ctx context.Context,
	key *settingsKeys,
) error {
	user, err := s.usersRepo.GetUser(ctx, key.username)
	if errors.Is(err, repository.ErrNoData) {
		return internal.ErrUnknownUser
	} else if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	key.userid = user.ID

	switch key.scope {
	case "device":
		device, err := s.devicesRepo.GetDevice(ctx, user.ID, key.devicename)
		if errors.Is(err, repository.ErrNoData) {
			return internal.ErrUnknownDevice
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		key.deviceid = &device.ID

	case "podcast":
		p, err := s.podcastsRepo.GetPodcast(ctx, user.ID, key.podcast)
		if errors.Is(err, repository.ErrNoData) {
			return internal.ErrUnknownPodcast
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		key.podcastid = &p.ID

	case "episode":
		p, err := s.podcastsRepo.GetPodcast(ctx, user.ID, key.podcast)
		if errors.Is(err, repository.ErrNoData) {
			return internal.ErrUnknownEpisode
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		key.podcastid = &p.ID

		e, err := s.episodesRepo.GetEpisode(ctx, user.ID, p.ID, key.episode)
		if errors.Is(err, repository.ErrNoData) {
			return internal.ErrUnknownPodcast
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		key.episodeid = &e.ID
	case "account":
		// no extra data
	default:
		return aerr.New("unknown scope").WithTag(aerr.ValidationError)
	}

	return nil
}

//------------------------------------------------------------------------------

type settingsKeys struct {
	username   string
	scope      string
	devicename string
	episode    string
	podcast    string

	userid    int64
	podcastid *int64
	deviceid  *int64
	episodeid *int64
}

func newSettingsKeysFromCmd(c *command.ChangeSettingsCmd) settingsKeys {
	return settingsKeys{
		username:   c.UserName,
		scope:      c.Scope,
		devicename: c.DeviceName,
		podcast:    c.Podcast,
		episode:    c.Episode,
	}
}

func newSettingsKeysFromQuery(q *query.SettingsQuery) settingsKeys {
	return settingsKeys{
		username:   q.UserName,
		scope:      q.Scope,
		devicename: q.DeviceName,
		podcast:    q.Podcast,
		episode:    q.Episode,
	}
}
