package service

//
// podcasts_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"testing"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestPodcastsServiceUserPodcasts(t *testing.T) {
	ctx, i := prepareTests(t)
	podcastsSrv := do.MustInvoke[*PodcastsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	subscribed := []string{"http://example.com/p1", "http://example.com/p2", "http://example.com/p3"}

	prepareTestSub(ctx, t, i, "user1", "dev1", subscribed...)

	podcasts, err := podcastsSrv.GetPodcasts(ctx, "user1")
	assert.NoErr(t, err)
	assert.Equal(t, len(podcasts), 3)
	assert.EqualSorted(t, model.PodcastsToUrls(podcasts), subscribed)

	podcasts, err = podcastsSrv.GetPodcasts(ctx, "user2")
	assert.NoErr(t, err)
	assert.Equal(t, len(podcasts), 0)

	podcasts, err = podcastsSrv.GetPodcasts(ctx, "user3")
	assert.ErrSpec(t, err, common.ErrUnknownUser)
}

func TestPodcastsServiceUserPodcastsExt(t *testing.T) {
	ctx, i := prepareTests(t)
	podcastsSrv := do.MustInvoke[*PodcastsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	subscribed := []string{"http://example.com/p1", "http://example.com/p2", "http://example.com/p3"}

	prepareTestSub(ctx, t, i, "user1", "dev1", subscribed...)

	podcasts, err := podcastsSrv.GetPodcastsWithLastEpisode(ctx, "user1", true)
	assert.NoErr(t, err)
	assert.Equal(t, len(podcasts), 3)
	assert.Equal(t, podcasts[0].URL, "http://example.com/p1")
	assert.Equal(t, podcasts[1].URL, "http://example.com/p2")
	assert.Equal(t, podcasts[2].URL, "http://example.com/p3")

	// TODO: check episode

	_, err = podcastsSrv.GetPodcastsWithLastEpisode(ctx, "user3", true)
	assert.ErrSpec(t, err, common.ErrUnknownUser)
}
