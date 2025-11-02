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
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var ErrUnknownPodcast = errors.New("unknown podcast")

type Podcasts struct {
	db           *db.Database
	usersRepo    repository.UsersRepository
	podcastsRepo repository.PodcastsRepository
}

func NewPodcastsServiceI(i do.Injector) (*Podcasts, error) {
	return &Podcasts{
		db:           do.MustInvoke[*db.Database](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
	}, nil
}

func (p *Podcasts) GetUserPodcasts(ctx context.Context, username string) ([]model.Podcast, error) {
	conn, err := p.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := p.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	subs, err := p.podcastsRepo.ListSubscribedPodcasts(ctx, conn, user.ID, time.Time{})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
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
