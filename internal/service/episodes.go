//
// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

type Episodes struct {
	repo *repository.Repository
}

func NewEpisodesService(repo *repository.Repository) *Episodes {
	return &Episodes{repo}
}

func (e *Episodes) SaveEpisodesActions(ctx context.Context, username string, action ...*model.Episode) error {
	user, err := e.repo.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return ErrUnknownUser
	}

	episodes := make([]*model.EpisodeDB, 0, len(action))

	for _, act := range action {
		episodes = append(episodes, &model.EpisodeDB{
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

	if err := e.repo.SaveEpisode(ctx, user.ID, episodes...); err != nil {
		return fmt.Errorf("save episodes error: %w", err)
	}

	return nil
}

func (e *Episodes) GetEpisodesActions(ctx context.Context, username, podcast, devicename string,
	since time.Time, aggregated bool,
) ([]*model.Episode, error) {
	user, err := e.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	var deviceid int64
	if devicename != "" {
		device, err := e.repo.GetDevice(ctx, user.ID, devicename)
		if err != nil {
			return nil, ErrUnknownDevice
		}

		deviceid = device.ID
	}

	var podcastid int64
	if podcast != "" {
		p, err := e.repo.GetPodcast(ctx, user.ID, podcast)
		if err != nil {
			return nil, ErrUnknownPodcast
		}

		podcastid = p.ID
	}

	episodes, err := e.repo.GetEpisodes(ctx, user.ID, deviceid, podcastid, since, aggregated)
	if err != nil {
		return nil, fmt.Errorf("get episodes error: %w", err)
	}

	res := make([]*model.Episode, 0, len(episodes))

	for _, e := range episodes {
		res = append(res, &model.Episode{
			Podcast:   e.PodcastURL,
			Device:    e.Device,
			Episode:   e.URL,
			Action:    e.Action,
			Timestamp: e.UpdatedAt,
			Started:   e.Started,
			Position:  e.Position,
			Total:     e.Total,
		})
	}

	return res, nil
}
