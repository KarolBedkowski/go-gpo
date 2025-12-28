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

			// TODO: settings

			res = append(res, esu)
		}

		return res, nil
	})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return res, nil
}
