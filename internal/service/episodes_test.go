package service

//
// episodes_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestEpisodesServiceSave(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*Episodes](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestSub(
		ctx,
		t,
		i,
		"user1",
		"dev1",
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	)

	started, position, total := 10, 20, 300
	episodeActions := []model.Episode{
		{
			Podcast:   "http://example.com/p1",
			Episode:   "http://example.com/p1/ep1",
			Device:    "dev1",
			Action:    "download",
			Timestamp: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			Podcast:   "http://example.com/p1",
			Episode:   "http://example.com/p1/ep1",
			Device:    "dev1",
			Timestamp: time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC),
			Action:    "play",
			Started:   &started,
			Position:  &position,
			Total:     &total,
		},
		{
			Podcast:   "http://example.com/p1",
			Episode:   "http://example.com/p1/ep2",
			Device:    "dev1",
			Action:    "download",
			Timestamp: time.Date(2025, 1, 4, 3, 4, 5, 0, time.UTC),
		},
		{
			Podcast:   "http://example.com/p2",
			Episode:   "http://example.com/p2/ep1",
			Device:    "dev2",
			Action:    "delete",
			Timestamp: time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC),
		},
	}

	err := episodesSrv.SaveEpisodesActions(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	// get last action for each episodes
	episodes, err := episodesSrv.GetPodcastEpisodes(ctx, "user1", "", "")
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	// list is sorted by updated_at
	assert.Equal(t, episodes[0].Podcast, "http://example.com/p1")
	assert.Equal(t, episodes[0].Episode, "http://example.com/p1/ep1")
	assert.Equal(t, episodes[0].Action, "play")
	assert.Equal(t, episodes[0].Device, "dev1")
	assert.Equal(t, episodes[0].Timestamp, time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, *episodes[0].Started, 10)
	assert.Equal(t, *episodes[0].Position, 20)
	assert.Equal(t, *episodes[0].Total, 300)
	assert.Equal(t, episodes[1].Podcast, "http://example.com/p1")
	assert.Equal(t, episodes[1].Episode, "http://example.com/p1/ep2")
	assert.Equal(t, episodes[1].Device, "dev1")
	assert.Equal(t, episodes[1].Action, "download")
	assert.Equal(t, episodes[1].Timestamp, time.Date(2025, 1, 4, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, episodes[1].Started, nil)
	assert.Equal(t, episodes[1].Position, nil)
	assert.Equal(t, episodes[1].Total, nil)
	assert.Equal(t, episodes[2].Podcast, "http://example.com/p2")
	assert.Equal(t, episodes[2].Episode, "http://example.com/p2/ep1")
	assert.Equal(t, episodes[2].Action, "delete")
	assert.Equal(t, episodes[2].Device, "dev2")
	assert.Equal(t, episodes[2].Timestamp, time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, episodes[2].Started, nil)
	assert.Equal(t, episodes[2].Position, nil)
	assert.Equal(t, episodes[2].Total, nil)
}
