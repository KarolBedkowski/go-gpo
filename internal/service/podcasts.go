package service

//
// podcasts.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var ErrUnknownPodcast = errors.New("unknown podcast")

type Podcasts struct {
	db *db.Database
}

func NewPodcastsService(db *db.Database) *Podcasts {
	return &Podcasts{db}
}

func (p *Podcasts) GetUserPodcasts(ctx context.Context, username string) ([]model.Podcast, error) {
	conn, err := p.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := p.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	subs, err := repo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	podcasts := make([]model.Podcast, 0, len(subs))

	for _, s := range subs {
		podcasts = append(podcasts, model.Podcast{
			Title: s.Title,
			URL:   s.URL,
		})
	}

	return podcasts, nil
}
