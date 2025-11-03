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
	"slices"
	"time"

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

func (e *Episodes) GetPodcastEpisodes(ctx context.Context, username, podcast, devicename string,
) ([]model.Episode, error) {
	conn, err := e.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	actions, err := e.getEpisodesActions(ctx, conn, username, podcast, devicename, time.Time{}, false)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if len(actions) == 0 {
		return []model.Episode{}, nil
	}

	// get last entry for each episode
	// TODO: move it to db
	slices.Reverse(actions)

	seen := make(map[string]struct{})
	episodes := make([]model.Episode, 0, len(actions))

	for _, episode := range actions {
		if _, ok := seen[episode.URL]; ok {
			continue
		}

		seen[episode.URL] = struct{}{}
		episodes = append(episodes, model.Episode{
			Episode:   episode.URL,
			Device:    episode.Device,
			Action:    episode.Action,
			Timestamp: episode.UpdatedAt,
			Started:   episode.Started,
			Position:  episode.Position,
			Total:     episode.Total,
			Podcast:   episode.PodcastURL,
		})
	}

	slices.Reverse(episodes)

	return episodes, nil
}

func (e *Episodes) SaveEpisodesActions(ctx context.Context, username string, action ...model.Episode) error {
	err := e.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		user, err := e.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		episodes := make([]repository.EpisodeDB, 0, len(action))

		for _, act := range action {
			episodes = append(episodes, repository.EpisodeDB{
				URL:        act.Episode,
				Device:     act.Device,
				Action:     act.Action,
				UpdatedAt:  act.Timestamp,
				CreatedAt:  act.Timestamp,
				Started:    act.Started,
				Position:   act.Position,
				Total:      act.Total,
				PodcastURL: act.Podcast,
			})
		}

		if err := e.episodesRepo.SaveEpisode(ctx, dbctx, user.ID, episodes...); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})

	return err //nolint:wrapcheck
}

func (e *Episodes) GetEpisodesActions(ctx context.Context, username, podcast, devicename string,
	since time.Time, aggregated bool,
) ([]model.Episode, error) {
	conn, err := e.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	episodes, err := e.getEpisodesActions(ctx, conn, username, podcast, devicename, since, aggregated)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	res := make([]model.Episode, 0, len(episodes))

	for _, e := range episodes {
		episode := model.Episode{
			Podcast:   e.PodcastURL,
			Device:    e.Device,
			Episode:   e.URL,
			Action:    e.Action,
			Timestamp: e.UpdatedAt,
		}
		if e.Action == "play" {
			episode.Started = e.Started
			episode.Position = e.Position
			episode.Total = e.Total
		}

		res = append(res, episode)
	}

	return res, nil
}

func (e *Episodes) GetEpisodesUpdates(ctx context.Context, username, devicename string, since time.Time,
	includeActions bool,
) ([]model.EpisodeUpdate, error) {
	_ = includeActions

	conn, err := e.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	episodes, err := e.getEpisodesActions(ctx, conn, username, "", devicename, since, true)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	res := make([]model.EpisodeUpdate, 0, len(episodes))

	for _, e := range episodes {
		ep := model.EpisodeUpdate{
			Title:        e.Title,
			URL:          e.URL,
			PodcastTitle: e.PodcastTitle,
			PodcastURL:   e.PodcastURL,
			Status:       e.Action,
			// do not tracking released time; use updated time
			Released: e.UpdatedAt,
		}
		res = append(res, ep)
	}

	return res, nil
}

func (e *Episodes) getEpisodesActions(
	ctx context.Context,
	dbctx repository.DBContext,
	username, podcast, devicename string,
	since time.Time,
	aggregated bool,
) ([]repository.EpisodeDB, error) {
	user, err := e.usersRepo.GetUser(ctx, dbctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	var deviceid int64

	if devicename != "" {
		device, err := e.devicesRepo.GetDevice(ctx, dbctx, user.ID, devicename)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownDevice
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		deviceid = device.ID
	}

	var podcastid int64

	if podcast != "" {
		p, err := e.podcastsRepo.GetPodcast(ctx, dbctx, user.ID, podcast)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownPodcast
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcastid = p.ID
	}

	episodes, err := e.episodesRepo.ListEpisodes(ctx, dbctx, user.ID, deviceid, podcastid, since, aggregated)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return episodes, nil
}
