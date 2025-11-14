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
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
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

	episodeActions := prepareEpisodes()
	err := episodesSrv.AddActiong(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	// get last action for each episodes
	episodes, err := episodesSrv.GetEpisodes(ctx, "user1", "", "")
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

	// only one podcast, device should be ignored
	episodes, err = episodesSrv.GetEpisodes(ctx, "user1", "dev2", "http://example.com/p1")
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
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
}

func TestEpisodesServiceActions(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
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

	episodeActions := prepareEpisodes()
	err := episodesSrv.AddActiong(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	// get all
	episodes, err := episodesSrv.GetActions(ctx, "user1", "", "", time.Time{}, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 4)
	assert.Equal(t, episodes[0], episodeActions[0])
	assert.Equal(t, episodes[1], episodeActions[1])
	assert.Equal(t, episodes[2], episodeActions[2])
	assert.Equal(t, episodes[3], episodeActions[3])

	// get since 2025-01-04
	episodes, err = episodesSrv.GetActions(ctx, "user1", "", "",
		time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), false,
	)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
	assert.Equal(t, episodes[0], episodeActions[2])
	assert.Equal(t, episodes[1], episodeActions[3])

	// get all aggregated (last action)
	episodes, err = episodesSrv.GetActions(ctx, "user1", "", "", time.Time{}, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	assert.Equal(t, episodes[0], episodeActions[1])
	assert.Equal(t, episodes[1], episodeActions[2])
	assert.Equal(t, episodes[2], episodeActions[3])

	// get one podcase aggregated; device should be ignored
	episodes, err = episodesSrv.GetActions(ctx, "user1", "http://example.com/p1", "dev2", time.Time{}, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	assert.Equal(t, episodes[0], episodeActions[0])
	assert.Equal(t, episodes[1], episodeActions[1])
	assert.Equal(t, episodes[2], episodeActions[2])

	// get one podcase aggregated; device should be ignored; aggregated
	episodes, err = episodesSrv.GetActions(ctx, "user1", "http://example.com/p1", "dev2", time.Time{}, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
	assert.Equal(t, episodes[0], episodeActions[1])
	assert.Equal(t, episodes[1], episodeActions[2])
}

func TestEpisodesServiceUpdates(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
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

	episodeActions := prepareEpisodes()
	err := episodesSrv.AddActiong(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	// without device, no include actions (last action only)
	updates, err := episodesSrv.GetUpdates(ctx, "user1", "", time.Time{}, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 3)
	assert.Equal(t, updates[0].URL, episodeActions[1].Episode)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[1].Podcast)
	assert.Equal(t, updates[0].Status, episodeActions[1].Action)
	assert.Equal(t, updates[0].Episode, nil)
	assert.Equal(t, updates[1].URL, episodeActions[2].Episode)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[2].Podcast)
	assert.Equal(t, updates[1].Status, episodeActions[2].Action)
	assert.Equal(t, updates[1].Episode, nil)
	assert.Equal(t, updates[2].URL, episodeActions[3].Episode)
	assert.Equal(t, updates[2].PodcastURL, episodeActions[3].Podcast)
	assert.Equal(t, updates[2].Status, episodeActions[3].Action)
	assert.Equal(t, updates[2].Episode, nil)

	// without device, include action
	updates, err = episodesSrv.GetUpdates(ctx, "user1", "",
		time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), true)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 2)
	assert.Equal(t, updates[0].URL, episodeActions[2].Episode)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[2].Podcast)
	assert.Equal(t, updates[0].Status, episodeActions[2].Action)
	assert.Equal(t, *updates[0].Episode, episodeActions[2])
	assert.Equal(t, updates[1].URL, episodeActions[3].Episode)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[3].Podcast)
	assert.Equal(t, updates[1].Status, episodeActions[3].Action)
	assert.Equal(t, *updates[1].Episode, episodeActions[3])

	// with device (should return entries other that with dev1), include action
	updates, err = episodesSrv.GetUpdates(ctx, "user1", "dev2",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), true)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 2)
	assert.Equal(t, updates[0].URL, episodeActions[1].Episode)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[1].Podcast)
	assert.Equal(t, updates[0].Status, episodeActions[1].Action)
	assert.Equal(t, *updates[0].Episode, episodeActions[1])
	assert.Equal(t, updates[1].URL, episodeActions[2].Episode)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[2].Podcast)
	assert.Equal(t, updates[1].Status, episodeActions[2].Action)
	assert.Equal(t, *updates[1].Episode, episodeActions[2])
}

func TestEpisodesServiceLastEpisodes(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
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

	episodeActions := prepareEpisodes()
	err := episodesSrv.AddActiong(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	actions, err := episodesSrv.GetLastActions(ctx, "user1",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 2)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 2)
	assert.Equal(t, actions[0].Episode, episodeActions[2].Episode)
	assert.Equal(t, actions[0].Podcast, episodeActions[2].Podcast)
	assert.Equal(t, actions[0].Action, episodeActions[2].Action)
	assert.Equal(t, actions[1].Episode, episodeActions[3].Episode)
	assert.Equal(t, actions[1].Podcast, episodeActions[3].Podcast)
	assert.Equal(t, actions[1].Action, episodeActions[3].Action)

	actions, err = episodesSrv.GetLastActions(ctx, "user1",
		time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), 0)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 3)
	assert.Equal(t, actions[0].Episode, episodeActions[1].Episode)
	assert.Equal(t, actions[0].Podcast, episodeActions[1].Podcast)
	assert.Equal(t, actions[0].Action, episodeActions[1].Action)
	assert.Equal(t, actions[1].Episode, episodeActions[2].Episode)
	assert.Equal(t, actions[1].Podcast, episodeActions[2].Podcast)
	assert.Equal(t, actions[1].Action, episodeActions[2].Action)
	assert.Equal(t, actions[2].Episode, episodeActions[3].Episode)
	assert.Equal(t, actions[2].Podcast, episodeActions[3].Podcast)
	assert.Equal(t, actions[2].Action, episodeActions[3].Action)

	actions, err = episodesSrv.GetLastActions(ctx, "user1",
		time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), 1)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 1)
	assert.Equal(t, actions[0].Episode, episodeActions[3].Episode)
	assert.Equal(t, actions[0].Podcast, episodeActions[3].Podcast)
	assert.Equal(t, actions[0].Action, episodeActions[3].Action)
}

func TestEpisodesServiceFavorites(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	settSrv := do.MustInvoke[*SettingsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
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

	episodeActions := prepareEpisodes()
	err := episodesSrv.AddActiong(ctx, "user1", episodeActions...)
	assert.NoErr(t, err)

	setkey := model.NewSettingsKey("user1", "episode", "dev1",
		episodeActions[1].Podcast, episodeActions[1].Episode)
	err = settSrv.SaveSettings(ctx, &setkey, map[string]string{"is_favorite": "true"})
	assert.NoErr(t, err)

	setkey = model.NewSettingsKey("user1", "episode", "dev1",
		episodeActions[3].Podcast, episodeActions[3].Episode)
	err = settSrv.SaveSettings(ctx, &setkey, map[string]string{"is_favorite": "true"})
	assert.NoErr(t, err)

	favs, err := episodesSrv.GetFavorites(ctx, "user1")
	assert.NoErr(t, err)
	assert.Equal(t, len(favs), 2)
	assert.Equal(t, favs[0].URL, episodeActions[1].Episode)
	assert.Equal(t, favs[1].URL, episodeActions[3].Episode)
}

func TestEpisodesServiceNewDevPodcast(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	ctx = log.Logger.WithContext(ctx)
	episodesSrv := do.MustInvoke[*EpisodesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
	prepareTestSub(
		ctx,
		t,
		i,
		"user1",
		"dev1",
		"http://example.com/p1",
	)

	action := model.Episode{
		Podcast:   "http://example.com/p2",
		Episode:   "http://example.com/p2/ep1",
		Device:    "dev3", // new device
		Action:    "delete",
		Timestamp: time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC),
	}

	err := episodesSrv.AddActiong(ctx, "user1", action)
	assert.NoErr(t, err)

	episodes, err := episodesSrv.GetEpisodes(ctx, "user1", "dev1", "")
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 1)
	assert.Equal(t, episodes[0].Device, "dev3")
	assert.Equal(t, episodes[0].Podcast, "http://example.com/p2")
}

func prepareEpisodes() []model.Episode {
	started, position, total := 10, 20, 300

	return []model.Episode{
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
}
