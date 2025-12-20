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
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type PodcastsSrv struct {
	db           *db.Database
	usersRepo    repository.Users
	podcastsRepo repository.Podcasts
	episodesRepo repository.Episodes
}

func NewPodcastsSrv(i do.Injector) (*PodcastsSrv, error) {
	return &PodcastsSrv{
		db:           do.MustInvoke[*db.Database](i),
		usersRepo:    do.MustInvoke[repository.Users](i),
		podcastsRepo: do.MustInvoke[repository.Podcasts](i),
		episodesRepo: do.MustInvoke[repository.Episodes](i),
	}, nil
}

func (p *PodcastsSrv) GetPodcast(ctx context.Context, username string, podcastid int64) (*model.Podcast, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.db, func(ctx context.Context) (*model.Podcast, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcast, err := p.podcastsRepo.GetPodcastByID(ctx, user.ID, podcastid)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownPodcast
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return podcast, nil
	})
}

func (p *PodcastsSrv) GetPodcasts(ctx context.Context, username string) ([]model.Podcast, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}
	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.db, func(ctx context.Context) ([]model.Podcast, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		subs, err := p.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return subs, nil
	})
}

func (p *PodcastsSrv) GetPodcastsWithLastEpisode(ctx context.Context, username string, subscribedOnly bool,
) ([]model.PodcastWithLastEpisode, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.db, func(ctx context.Context) ([]model.PodcastWithLastEpisode, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		var subs model.Podcasts
		if subscribedOnly {
			subs, err = p.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
		} else {
			subs, err = p.podcastsRepo.ListPodcasts(ctx, user.ID, time.Time{})
		}

		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcasts := make([]model.PodcastWithLastEpisode, len(subs))
		for idx, s := range subs {
			podcasts[idx] = model.PodcastWithLastEpisode{
				PodcastID:   s.ID,
				Title:       s.Title,
				URL:         s.URL,
				Website:     s.Website,
				Description: s.Description,
				Subscribed:  s.Subscribed,
			}

			lastEpisode, err := p.episodesRepo.GetLastEpisodeAction(ctx, user.ID, s.ID, false)
			if errors.Is(err, common.ErrNoData) {
				continue
			} else if err != nil {
				return nil, aerr.ApplyFor(ErrRepositoryError, err, "failed to get last episode")
			}

			podcasts[idx].LastEpisode = lastEpisode
		}

		return podcasts, nil
	})
}

//------------------------------------------------------------------------------

func (p *PodcastsSrv) DeletePodcast(ctx context.Context, username string, podcastid int64) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Int64("podcast_id", podcastid).Msg("delete podcast")

	//nolint:wrapcheck
	return db.InTransaction(ctx, p.db, func(ctx context.Context) error {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		podcast, err := p.podcastsRepo.GetPodcastByID(ctx, user.ID, podcastid)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownPodcast
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return p.podcastsRepo.DeletePodcast(ctx, podcast.ID)
	})
}

//------------------------------------------------------------------------------

func (p *PodcastsSrv) DownloadPodcastsInfo(ctx context.Context, since time.Time) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msgf("start downloading podcasts info; since=%s", since)

	// get podcasts to update
	urls, err := db.InConnectionR(ctx, p.db, func(ctx context.Context) ([]string, error) {
		return p.podcastsRepo.ListPodcastsToUpdate(ctx, since)
	})
	if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	if len(urls) == 0 {
		logger.Debug().Msg("start downloading podcasts finished; no url to update found")

		return nil
	}

	logger.Debug().Msgf("start downloading podcasts finished; found %d", len(urls))

	tasks := make(chan string, len(urls))

	var wg sync.WaitGroup
	for range min(len(urls), 5) { //nolint:mnd
		wg.Go(func() { p.downloadPodcastInfoWorker(ctx, tasks) })
	}

	for _, u := range urls {
		tasks <- u
	}

	close(tasks)

	wg.Wait()

	logger.Info().Msgf("downloading podcasts info finished, count: %d", len(urls))

	return nil
}

const downloadPodcastInfoTimeout = 10 * time.Second

func (p *PodcastsSrv) downloadPodcastInfoWorker(ctx context.Context, urls <-chan string) {
	logger := zerolog.Ctx(ctx)

	for url := range urls {
		logger.Debug().Str("podcast_url", url).Msg("downloading podcast info")

		dctx, cancel := context.WithTimeout(ctx, downloadPodcastInfoTimeout)
		fp := gofeed.NewParser()
		feed, err := fp.ParseURLWithContext(url, dctx)

		cancel()

		if err != nil {
			logger.Info().Str("podcast_url", url).Err(err).Msg("download podcast info failed")

			continue
		}

		logger.Debug().Str("podcast_url", url).Msgf("got podcast title: %q", feed.Title)

		title := feed.Title
		if title == "" {
			title = "<no title>"
		}

		update := model.PodcastMetaUpdate{
			URL:           url,
			Title:         title,
			Description:   feed.Description,
			Website:       feed.Link,
			MetaUpdatedAt: time.Now().UTC(),
		}

		err = db.InTransaction(ctx, p.db, func(ctx context.Context) error {
			return p.podcastsRepo.UpdatePodcastsInfo(ctx, &update)
		})
		if err != nil {
			logger.Error().Err(err).Str("podcast_url", url).Msg("update podcast info failed")
		}
	}
}
