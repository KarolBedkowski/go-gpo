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
	"net/http"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type PodcastsSrv struct {
	dbi          repository.Database
	usersRepo    repository.Users
	podcastsRepo repository.Podcasts
	episodesRepo repository.Episodes
}

func NewPodcastsSrv(i do.Injector) (*PodcastsSrv, error) {
	return &PodcastsSrv{
		dbi:          do.MustInvoke[repository.Database](i),
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
	return db.InConnectionR(ctx, p.dbi, func(ctx context.Context) (*model.Podcast, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcast: user loaded")

		podcast, err := p.podcastsRepo.GetPodcastByID(ctx, user.ID, podcastid)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownPodcast
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcast: podcast loaded")

		return podcast, nil
	})
}

func (p *PodcastsSrv) GetPodcasts(ctx context.Context, username string) ([]model.Podcast, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}
	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.dbi, func(ctx context.Context) ([]model.Podcast, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcasts: user loaded")

		subs, err := p.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcasts: podcasts loaded")

		return subs, nil
	})
}

func (p *PodcastsSrv) GetPodcastsWithLastEpisode(ctx context.Context, username string, subscribedOnly bool,
) ([]model.PodcastWithLastEpisode, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, p.dbi, func(ctx context.Context) ([]model.PodcastWithLastEpisode, error) {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcastsWithLastEpisode: user loaded")

		var subs model.Podcasts
		if subscribedOnly {
			subs, err = p.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, time.Time{})
		} else {
			subs, err = p.podcastsRepo.ListPodcasts(ctx, user.ID, time.Time{})
		}

		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "GetPodcastsWithLastEpisode: podcasts loaded")

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

		common.TraceLazyPrintf(ctx, "GetPodcastsWithLastEpisode: model prepared")

		return podcasts, nil
	})
}

//------------------------------------------------------------------------------

func (p *PodcastsSrv) DeletePodcast(ctx context.Context, username string, podcastid int64) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Int64("podcast_id", podcastid).
		Msgf("PodcastsSrv: delete podcast user_name=%s podcast_id=%d", username, podcastid)

	//nolint:wrapcheck
	return db.InTransaction(ctx, p.dbi, func(ctx context.Context) error {
		user, err := p.usersRepo.GetUser(ctx, username)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "DeletePodcast: user loaded")

		podcast, err := p.podcastsRepo.GetPodcastByID(ctx, user.ID, podcastid)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownPodcast
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "DeletePodcast: podcast loaded")

		err = p.podcastsRepo.DeletePodcast(ctx, podcast.ID)

		common.TraceLazyPrintf(ctx, "DeletePodcast: podcast deleted")

		return err
	})
}

//------------------------------------------------------------------------------

func (p *PodcastsSrv) ResolvePodcastsURL(ctx context.Context, urls []string) map[string]model.ResolvedPodcastURL {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Strs("urls", urls).Msgf("PodcastsSrv: start resolving podcasts url")

	if len(urls) == 0 {
		return nil
	}

	tasks := make(chan string, len(urls))
	res := make(chan model.ResolvedPodcastURL, len(urls))

	var wg sync.WaitGroup
	for range min(len(urls), 5) { //nolint:mnd
		wg.Go(func() {
			resolvePodcastsURLTask(ctx, tasks, res)
		})
	}

	for _, u := range urls {
		tasks <- u
	}

	close(tasks)

	wg.Wait()

	close(res)

	resolved := make(map[string]model.ResolvedPodcastURL, len(urls))
	for r := range res {
		resolved[r.URL] = r
	}

	logger.Info().Msgf("PodcastsSrv: resolving podcasts url finished, count=%d", len(urls))

	return resolved
}

//------------------------------------------------------------------------------

func (p *PodcastsSrv) DownloadPodcastsInfo(ctx context.Context, since time.Time, loadepisodes bool) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msgf("PodcastsSrv: start downloading podcasts info; since=%s", since)

	// get podcasts to update
	urls, err := db.InConnectionR(ctx, p.dbi, func(ctx context.Context) ([]model.PodcastToUpdate, error) {
		return p.podcastsRepo.ListPodcastsToUpdate(ctx, since)
	})
	if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	if len(urls) == 0 {
		logger.Debug().Msg("PodcastsSrv: download podcasts finished; no url to update found")

		return nil
	}

	logger.Debug().Msgf("PodcastsSrv: podcast_to_update=%d", len(urls))

	eventlog := common.NewEventLog("srv.podcasts", "downloadPodcastInfo")
	defer eventlog.Close()

	tasks := make(chan model.PodcastToUpdate, len(urls))

	var wg sync.WaitGroup
	for range min(len(urls), 5) { //nolint:mnd
		wg.Go(func() { p.downloadPodcastInfoWorker(ctx, tasks, since, loadepisodes, eventlog) })
	}

	for _, u := range urls {
		tasks <- u
	}

	close(tasks)

	wg.Wait()

	logger.Info().Msg("PodcastsSrv: downloading podcasts info finished")

	return nil
}

const downloadPodcastInfoTimeout = 10 * time.Second

func (p *PodcastsSrv) downloadPodcastInfoWorker(
	ctx context.Context, tasks <-chan model.PodcastToUpdate, since time.Time, loadepisodes bool,
	eventlog *common.EventLog,
) {
	logger := zerolog.Ctx(ctx)
	if logger == nil {
		panic("no logger in ctx")
	}

	fp := gofeed.NewParser()
	fp.UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:146.0) Gecko/20100101 Firefox/146.0"

	for task := range tasks {
		taskid := xid.New()
		llogger := logger.With().Str("podcast_url", task.URL).Str("taskid", taskid.String()).Logger()
		lctx := hlog.CtxWithID(llogger.WithContext(ctx), taskid)

		eventlog.Printf("processing %q", task.URL)

		err := p.downloadPodcastInfo(lctx, fp, since, &task, loadepisodes, eventlog)
		if err != nil {
			eventlog.Errorf("update failed: error=%q", err)
			llogger.Warn().Err(err).Msgf("PodcastsSrv: update podcast_url=%q error=%q", task.URL, err)
		} else {
			eventlog.Errorf("update %q finished", task.URL)
		}
	}
}

func (p *PodcastsSrv) downloadPodcastInfo(ctx context.Context, //nolint: cyclop
	feedparser *gofeed.Parser, since time.Time, task *model.PodcastToUpdate, loadepisodes bool,
	eventlog *common.EventLog,
) error {
	logger := zerolog.Ctx(ctx)
	if logger == nil {
		panic("missing logger in ctx")
	}

	logger.Debug().Msgf("PodcastsSrv: downloading podcast_url=%q", task.URL)

	var (
		update   model.PodcastMetaUpdate
		episodes []model.Episode
	)

	feed, status, err := parseFeedURLWithContext(ctx, feedparser, task)
	eventlog.Printf("download %q got status=%q error=%q", task.URL, status, err)

	switch {
	case err != nil:
		return err
	case status == http.StatusNotModified:
		logger.Debug().Err(err).Msgf("PodcastsSrv: podcast_url=%q not modified", task.URL)

		update = model.PodcastMetaUpdate{URL: task.URL, MetaUpdatedAt: time.Now().UTC(), NotModified: true}
	case feed != nil:
		logger.Debug().Msgf("PodcastsSrv: for podcast_url=%q got podcast title=%q published=%s updated=%s",
			task.URL, feed.Title, feed.UpdatedParsed, feed.PublishedParsed)

		if !feedNeedToBeUpdated(feed, task.MetaUpdatedAt) {
			logger.Debug().Msgf("PodcastsSrv: podcast_url=%q not updated, skipping", task.URL)

			return nil
		}

		update = podcastToUpdate(task.URL, feed)
		if loadepisodes {
			episodes = episodesToUpdate(feed, since, task.MetaUpdatedAt)
		}
	default:
		logger.Info().Int("status_code", status).
			Msgf("PodcastsSrv: download podcast_url=%q unknown status=%d", task.URL, status)

		return nil
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, p.dbi, func(ctx context.Context) error {
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
}

func parseFeedURLWithContext(ctx context.Context, feedparser *gofeed.Parser, //nolint:cyclop
	ptu *model.PodcastToUpdate,
) (*gofeed.Feed, int, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadPodcastInfoTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ptu.URL, nil)
	if err != nil {
		return nil, 0, aerr.Wrapf(err, "create request failed")
	}

	if !ptu.MetaUpdatedAt.IsZero() {
		req.Header.Add("If-Modified-Since", ptu.MetaUpdatedAt.Format(time.RFC1123))
	}

	req.Header.Set("User-Agent", feedparser.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, aerr.Wrapf(err, "make request failed")
	} else if resp == nil {
		return nil, 0, aerr.New("empty response when get feed")
	}

	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		return nil, http.StatusNotModified, nil
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		res, err := feedparser.Parse(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, aerr.Wrapf(err, "parse feed body failed")
		}

		return res, resp.StatusCode, nil
	case resp.StatusCode >= 400: //nolint:mnd
		return nil, resp.StatusCode, aerr.New("invalid response from when get feed").
			WithMeta("status_code", resp.StatusCode, "status", resp.Status)
	default:
		return nil, resp.StatusCode, nil
	}
}

func podcastToUpdate(url string, feed *gofeed.Feed) model.PodcastMetaUpdate {
	title := feed.Title
	if title == "" {
		title = "<no title>"
	}

	return model.PodcastMetaUpdate{
		URL:           url,
		Title:         title,
		Description:   feed.Description,
		Website:       feed.Link,
		MetaUpdatedAt: time.Now().UTC(),
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
		if u := validators.SanitizeURL(e.URL); u != "" {
			return u
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

func resolvePodcastsURLTask(
	ctx context.Context, urls <-chan string, res chan model.ResolvedPodcastURL,
) {
	tlogger := zerolog.Ctx(ctx)
	if tlogger == nil {
		panic("no logger in ctx")
	}

	for url := range urls {
		logger := tlogger.With().Str("podcast_url", url).Logger()
		logger.Debug().Msg("PodcastsSrv: downloading podcast info")

		dctx, cancel := context.WithTimeout(ctx, downloadPodcastInfoTimeout)
		resolvedurl, err := ResolvePodcastURL(dctx, url)

		cancel()

		res <- model.ResolvedPodcastURL{
			URL:         url,
			ResolvedURL: resolvedurl,
			Err:         err,
		}
	}
}

func ResolvePodcastURL(ctx context.Context, url string) (string, error) {
	client := http.DefaultClient

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return url, aerr.Wrapf(err, "create request error").WithTag(aerr.InternalError).WithMeta("url", url)
	}

	resp, err := client.Do(req)
	if err != nil {
		return url, aerr.Wrapf(err, "request failed").WithTag(aerr.InternalError).WithMeta("url", url)
	}

	if resp == nil {
		return url, aerr.New("request failed; empty resp").WithTag(aerr.InternalError).WithMeta("url", url)
	}

	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		if loc := resp.Header.Get("Location"); loc != "" {
			return loc, nil
		}

		return url, aerr.New("get location failed").WithMeta("url", url, "headers", resp.Header)
	}

	if resp.StatusCode == http.StatusOK {
		return url, nil
	}

	return url, aerr.New("invalid response when resolving url %q: %d (%q)", url, resp.StatusCode, resp.Status).
		WithMeta("url", url, "headers", resp.Header)
}
