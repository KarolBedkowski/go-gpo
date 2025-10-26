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
	"fmt"
	"time"

	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

type Episodes struct {
	repo *repository.Database
}

func NewEpisodesService(repo *repository.Database) *Episodes {
	return &Episodes{repo}
}

func (e *Episodes) SaveEpisodesActions(ctx context.Context, username string, action ...model.Episode) error {
	err := e.repo.InTransaction(ctx, func(db repository.DBContext) error {
		repo := e.repo.GetRepository(db)

		user, err := repo.GetUser(ctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return fmt.Errorf("get user error: %w", err)
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

		if err := repo.SaveEpisode(ctx, user.ID, episodes...); err != nil {
			return fmt.Errorf("save episodes error: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("save episodes error: %w", err)
	}

	return nil
}

func (e *Episodes) GetEpisodesActions(ctx context.Context, username, podcast, devicename string,
	since time.Time, aggregated bool,
) ([]model.Episode, error) {
	conn, err := e.repo.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := e.repo.GetRepository(conn)

	episodes, err := e.getEpisodesActions(ctx, repo, username, podcast, devicename, since, aggregated)
	if err != nil {
		return nil, fmt.Errorf("get episodes error: %w", err)
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

	conn, err := e.repo.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	repo := e.repo.GetRepository(conn)

	episodes, err := e.getEpisodesActions(ctx, repo, username, "", devicename, since, true)
	if err != nil {
		return nil, fmt.Errorf("get episodes error: %w", err)
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
	repo repository.Repository,
	username, podcast, devicename string,
	since time.Time,
	aggregated bool,
) ([]repository.EpisodeDB, error) {
	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	var deviceid int64

	if devicename != "" {
		device, err := repo.GetDevice(ctx, user.ID, devicename)
		if err != nil {
			return nil, ErrUnknownDevice
		}

		deviceid = device.ID
	}

	var podcastid int64

	if podcast != "" {
		p, err := repo.GetPodcast(ctx, user.ID, podcast)
		if err != nil {
			return nil, ErrUnknownPodcast
		}

		podcastid = p.ID
	}

	episodes, err := repo.ListEpisodes(ctx, user.ID, deviceid, podcastid, since, aggregated)
	if err != nil {
		return nil, fmt.Errorf("get episodes error: %w", err)
	}

	return episodes, nil
}
