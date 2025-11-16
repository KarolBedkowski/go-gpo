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

type PodcastsSrv struct {
	db           *db.Database
	usersRepo    repository.UsersRepository
	podcastsRepo repository.PodcastsRepository
	episodesRepo repository.EpisodesRepository
}

func NewPodcastsSrv(i do.Injector) (*PodcastsSrv, error) {
	return &PodcastsSrv{
		db:           do.MustInvoke[*db.Database](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
		episodesRepo: do.MustInvoke[repository.EpisodesRepository](i),
	}, nil
}

func (p *PodcastsSrv) GetPodcasts(ctx context.Context, username string) ([]model.Podcast, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	subs, err := db.InConnectionR(ctx, p.db, func(dbctx repository.DBContext) ([]repository.PodcastDB, error) {
		user, err := p.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		subs, err := p.podcastsRepo.ListSubscribedPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return subs, nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
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

func (p *PodcastsSrv) GetPodcastsWithLastEpisode(ctx context.Context, username string,
) ([]model.PodcastWithLastEpisode, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.db, func(dbctx repository.DBContext) ([]model.PodcastWithLastEpisode, error) {
		user, err := p.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		subs, err := p.podcastsRepo.ListSubscribedPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcasts := make([]model.PodcastWithLastEpisode, len(subs))
		for idx, s := range subs {
			podcasts[idx] = model.PodcastWithLastEpisode{
				Title: s.Title,
				URL:   s.URL,
			}

			lastEpisode, err := p.episodesRepo.GetLastEpisodeAction(ctx, dbctx, user.ID, s.ID, false)
			if errors.Is(err, repository.ErrNoData) {
				continue
			} else if err != nil {
				return nil, aerr.ApplyFor(ErrRepositoryError, err, "failed to get last episode")
			}

			ep := model.NewEpisodeFromDBModel(&lastEpisode)
			podcasts[idx].LastEpisode = &ep
		}

		return podcasts, nil
	})
}
