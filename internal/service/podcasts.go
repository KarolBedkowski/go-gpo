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
	urls, err := db.InConnectionR(ctx, p.db, func(ctx context.Context) ([]model.PodcastToUpdate, error) {
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

	tasks := make(chan model.PodcastToUpdate, len(urls))

	var wg sync.WaitGroup
	for range min(len(urls), 5) { //nolint:mnd
		wg.Go(func() { p.downloadPodcastInfoWorker(ctx, tasks, since) })
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

func (p *PodcastsSrv) downloadPodcastInfoWorker(
	ctx context.Context, tasks <-chan model.PodcastToUpdate, since time.Time,
) {
	tlogger := zerolog.Ctx(ctx)
	fp := gofeed.NewParser()

	for task := range tasks {
		logger := tlogger.With().Str("podcast_url", task.URL).Logger()
		logger.Debug().Msg("downloading podcast info")

		dctx, cancel := context.WithTimeout(ctx, downloadPodcastInfoTimeout)
		feed, err := fp.ParseURLWithContext(task.URL, dctx)

		cancel()

		if err != nil {
			logger.Info().Err(err).Msg("download podcast info failed")

			continue
		}

		logger.Debug().Msgf("got podcast title: %q; published: %s, updated: %s",
			feed.Title, feed.UpdatedParsed, feed.PublishedParsed)

		if !feedNeedToBeUpdated(feed, task.MetaUpdatedAt) {
			logger.Debug().Msg("not updated, skipping")

			continue
		}

		title := feed.Title
		if title == "" {
			title = "<no title>"
		}

		update := model.PodcastMetaUpdate{
			URL:           task.URL,
			Title:         title,
			Description:   feed.Description,
			Website:       feed.Link,
			MetaUpdatedAt: time.Now().UTC(),
		}
		episodes := episodesToUpdate(feed, since, task.MetaUpdatedAt)

		err = db.InTransaction(ctx, p.db, func(ctx context.Context) error {
			if err := p.podcastsRepo.UpdatePodcastsInfo(ctx, &update); err != nil {
				return aerr.Wrapf(err, "update podcast info failed")
			}

			if len(episodes) > 0 {
				if err := p.episodesRepo.UpdateEpisodeInfo(ctx, episodes...); err != nil {
					return aerr.Wrapf(err, "update episodes info failed")
				}
			}

			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("update podcast info failed")
		}
	}
}

func episodesToUpdate(feed *gofeed.Feed, since, metadataUpdatedAt time.Time) []model.Episode {
	episodes := make([]model.Episode, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item.Title != "" && itemNeedToBeUpdated(item, since, metadataUpdatedAt) {
			if url := findEpisodeURL(item); url != "" {
				episodes = append(episodes, model.Episode{
					Title: item.Title,
					GUID:  &item.GUID,
					URL:   url,
				})
			}
		}
	}

	return episodes
}

func findEpisodeURL(item *gofeed.Item) string {
	for _, e := range item.Enclosures {
		if e.URL != "" {
			return e.URL
		}
	}

	return ""
}

func feedNeedToBeUpdated(feed *gofeed.Feed, since time.Time) bool {
	if since.IsZero() {
		return true
	}

	if feed.PublishedParsed != nil && feed.PublishedParsed.After(since) {
		return true
	}

	if feed.UpdatedParsed != nil && feed.UpdatedParsed.After(since) {
		return true
	}

	// if no update and publish date - update
	return feed.UpdatedParsed == nil && feed.PublishedParsed == nil
}

func itemNeedToBeUpdated(item *gofeed.Item, since, metadataUpdatedAt time.Time) bool {
	// if podcast was updated - load episodes only from last update. Otherwise load
	// episodes `since`.
	if !metadataUpdatedAt.IsZero() {
		since = metadataUpdatedAt
	}

	if item.PublishedParsed != nil && item.PublishedParsed.After(since) {
		return true
	}

	if item.UpdatedParsed != nil && item.UpdatedParsed.After(since) {
		return true
	}

	// if no update and publish date - update
	return item.UpdatedParsed == nil && item.PublishedParsed == nil
}
