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

	podcasts, err := e.repo.GetPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get podcasts error: %w", err)
	}

	podcastsmap := podcasts.ToMap()
	episodes := make([]*model.EpisodeDB, 0, len(action))

	for _, act := range action {
		p, ok := podcastsmap[act.Podcast]
		if !ok {
			p = &model.PodcastDB{UserID: user.ID, URL: act.Podcast, Subscribed: true}
		}

		episodes = append(episodes, &model.EpisodeDB{
			PodcastID: p.ID,
			URL:       act.Episode,
			Action:    act.Action,
			UpdatedAt: act.Timestamp,
			Started:   act.Started,
			Position:  act.Position,
			Total:     act.Total,
			Podcast:   p,
		})
	}

	if err := e.repo.SaveEpisode(ctx, episodes...); err != nil {
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

	episodes, err := e.repo.GetEpisodes(ctx, user.ID, 0, since)
	if err != nil {
		return nil, fmt.Errorf("get episodes error: %w", err)
	}

	res := make([]*model.Episode, 0, len(episodes))

	for _, e := range episodes {
		res = append(res, &model.Episode{
			Podcast:   e.PodcastURL,
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
