package service

//
// episodes_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"testing"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
)

func TestEpisodesServiceSave(t *testing.T) {
	ctx, i := prepareTests(t)
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

	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: episodeActions})
	assert.NoErr(t, err)

	// get last action for each episodes
	q := query.GetEpisodesQuery{
		UserName:   "user1",
		Aggregated: true,
	}
	episodes, err := episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	// list is sorted by updated_at
	assert.Equal(t, episodes[0].Podcast.URL, "http://example.com/p1")
	assert.Equal(t, episodes[0].URL, "http://example.com/p1/ep1")
	assert.Equal(t, episodes[0].Action, "play")
	assert.Equal(t, episodes[0].Device.Name, "dev1")
	assert.Equal(t, episodes[0].Timestamp, time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, *episodes[0].Started, 10)
	assert.Equal(t, *episodes[0].Position, 20)
	assert.Equal(t, *episodes[0].Total, 300)
	assert.Equal(t, episodes[1].Podcast.URL, "http://example.com/p1")
	assert.Equal(t, episodes[1].URL, "http://example.com/p1/ep2")
	assert.Equal(t, episodes[1].Device.Name, "dev1")
	assert.Equal(t, episodes[1].Action, "download")
	assert.Equal(t, episodes[1].Timestamp, time.Date(2025, 1, 4, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, episodes[1].Started, nil)
	assert.Equal(t, episodes[1].Position, nil)
	assert.Equal(t, episodes[1].Total, nil)
	assert.Equal(t, episodes[2].Podcast.URL, "http://example.com/p2")
	assert.Equal(t, episodes[2].URL, "http://example.com/p2/ep1")
	assert.Equal(t, episodes[2].Action, "delete")
	assert.Equal(t, episodes[2].Device.Name, "dev2")
	assert.Equal(t, episodes[2].Timestamp, time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, episodes[2].Started, nil)
	assert.Equal(t, episodes[2].Position, nil)
	assert.Equal(t, episodes[2].Total, nil)

	// only one podcast, device should be ignored
	q = query.GetEpisodesQuery{
		UserName:   "user1",
		DeviceName: "dev2",
		Podcast:    "http://example.com/p1",
		Aggregated: true,
	}
	episodes, err = episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
	assert.Equal(t, episodes[0].Podcast.URL, "http://example.com/p1")
	assert.Equal(t, episodes[0].URL, "http://example.com/p1/ep1")
	assert.Equal(t, episodes[0].Action, "play")
	assert.Equal(t, episodes[0].Device.Name, "dev1")
	assert.Equal(t, episodes[0].Timestamp, time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, *episodes[0].Started, 10)
	assert.Equal(t, *episodes[0].Position, 20)
	assert.Equal(t, *episodes[0].Total, 300)
	assert.Equal(t, episodes[1].Podcast.URL, "http://example.com/p1")
	assert.Equal(t, episodes[1].URL, "http://example.com/p1/ep2")
	assert.Equal(t, episodes[1].Device.Name, "dev1")
	assert.Equal(t, episodes[1].Action, "download")
	assert.Equal(t, episodes[1].Timestamp, time.Date(2025, 1, 4, 3, 4, 5, 0, time.UTC))
	assert.Equal(t, episodes[1].Started, nil)
	assert.Equal(t, episodes[1].Position, nil)
	assert.Equal(t, episodes[1].Total, nil)
}

func TestEpisodesServiceActions(t *testing.T) {
	ctx, i := prepareTests(t)
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
	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: episodeActions})
	assert.NoErr(t, err)

	// get all
	q := query.GetEpisodesQuery{
		UserName: "user1",
	}
	episodes, err := episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 4)
	compareEpisodes(t, episodes[0], episodeActions[0])
	compareEpisodes(t, episodes[1], episodeActions[1])
	compareEpisodes(t, episodes[2], episodeActions[2])
	compareEpisodes(t, episodes[3], episodeActions[3])

	// get since 2025-01-04
	q = query.GetEpisodesQuery{
		UserName: "user1",
		Since:    time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
	}
	episodes, err = episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
	compareEpisodes(t, episodes[0], episodeActions[2])
	compareEpisodes(t, episodes[1], episodeActions[3])

	// get all aggregated (last action)
	q = query.GetEpisodesQuery{
		UserName:   "user1",
		Aggregated: true,
	}
	episodes, err = episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	compareEpisodes(t, episodes[0], episodeActions[1])
	compareEpisodes(t, episodes[1], episodeActions[2])
	compareEpisodes(t, episodes[2], episodeActions[3])

	// get one podcase aggregated; device should be ignored
	q = query.GetEpisodesQuery{
		UserName:   "user1",
		Podcast:    "http://example.com/p1",
		DeviceName: "dev2",
	}
	episodes, err = episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 3)
	compareEpisodes(t, episodes[0], episodeActions[0])
	compareEpisodes(t, episodes[1], episodeActions[1])
	compareEpisodes(t, episodes[2], episodeActions[2])

	// get one podcase aggregated; device should be ignored; aggregated
	q = query.GetEpisodesQuery{
		UserName:   "user1",
		Podcast:    "http://example.com/p1",
		DeviceName: "dev2",
		Aggregated: true,
	}
	episodes, err = episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 2)
	compareEpisodes(t, episodes[0], episodeActions[1])
	compareEpisodes(t, episodes[1], episodeActions[2])
}

func TestEpisodesServiceUpdates(t *testing.T) {
	ctx, i := prepareTests(t)
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
	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: episodeActions})
	assert.NoErr(t, err)

	// without device, no include actions (last action only)
	q := query.GetEpisodeUpdatesQuery{
		UserName: "user1",
	}
	updates, err := episodesSrv.GetUpdates(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 3)
	assert.Equal(t, updates[0].URL, episodeActions[1].URL)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[1].Podcast.URL)
	assert.Equal(t, updates[0].Status, episodeActions[1].Action)
	assert.Equal(t, updates[0].Episode, nil)
	assert.Equal(t, updates[1].URL, episodeActions[2].URL)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[2].Podcast.URL)
	assert.Equal(t, updates[1].Status, episodeActions[2].Action)
	assert.Equal(t, updates[1].Episode, nil)
	assert.Equal(t, updates[2].URL, episodeActions[3].URL)
	assert.Equal(t, updates[2].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, updates[2].Status, episodeActions[3].Action)
	assert.Equal(t, updates[2].Episode, nil)

	// without device, include action
	q = query.GetEpisodeUpdatesQuery{
		UserName:       "user1",
		Since:          time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
		IncludeActions: true,
	}
	updates, err = episodesSrv.GetUpdates(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 2)
	assert.Equal(t, updates[0].URL, episodeActions[2].URL)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[2].Podcast.URL)
	assert.Equal(t, updates[0].Status, episodeActions[2].Action)
	compareEpisodes(t, *updates[0].Episode, episodeActions[2])
	assert.Equal(t, updates[1].URL, episodeActions[3].URL)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, updates[1].Status, episodeActions[3].Action)
	compareEpisodes(t, *updates[1].Episode, episodeActions[3])

	// with device (should return all actions), include action
	q = query.GetEpisodeUpdatesQuery{
		UserName:       "user1",
		DeviceName:     "dev2",
		Since:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		IncludeActions: true,
	}
	updates, err = episodesSrv.GetUpdates(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(updates), 3)
	assert.Equal(t, updates[0].URL, episodeActions[1].URL)
	assert.Equal(t, updates[0].PodcastURL, episodeActions[1].Podcast.URL)
	assert.Equal(t, updates[0].Status, episodeActions[1].Action)
	compareEpisodes(t, *updates[0].Episode, episodeActions[1])
	assert.Equal(t, updates[1].URL, episodeActions[2].URL)
	assert.Equal(t, updates[1].PodcastURL, episodeActions[2].Podcast.URL)
	assert.Equal(t, updates[1].Status, episodeActions[2].Action)
	compareEpisodes(t, *updates[1].Episode, episodeActions[2])
	assert.Equal(t, updates[2].URL, episodeActions[3].URL)
	assert.Equal(t, updates[2].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, updates[2].Status, episodeActions[3].Action)
	compareEpisodes(t, *updates[2].Episode, episodeActions[3])
}

func TestEpisodesServiceLastEpisodes(t *testing.T) {
	ctx, i := prepareTests(t)
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
	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: episodeActions})
	assert.NoErr(t, err)

	q := query.GetLastEpisodesActionsQuery{
		UserName: "user1",
		Since:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Limit:    2,
	}
	actions, err := episodesSrv.GetLastActions(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 2)
	assert.Equal(t, actions[0].Episode, episodeActions[2].URL)
	assert.Equal(t, actions[0].PodcastURL, episodeActions[2].Podcast.URL)
	assert.Equal(t, actions[0].Action, episodeActions[2].Action)
	assert.Equal(t, actions[1].Episode, episodeActions[3].URL)
	assert.Equal(t, actions[1].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, actions[1].Action, episodeActions[3].Action)

	q = query.GetLastEpisodesActionsQuery{
		UserName: "user1",
		Since:    time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	actions, err = episodesSrv.GetLastActions(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 3)
	assert.Equal(t, actions[0].Episode, episodeActions[1].URL)
	assert.Equal(t, actions[0].PodcastURL, episodeActions[1].Podcast.URL)
	assert.Equal(t, actions[0].Action, episodeActions[1].Action)
	assert.Equal(t, actions[1].Episode, episodeActions[2].URL)
	assert.Equal(t, actions[1].PodcastURL, episodeActions[2].Podcast.URL)
	assert.Equal(t, actions[1].Action, episodeActions[2].Action)
	assert.Equal(t, actions[2].Episode, episodeActions[3].URL)
	assert.Equal(t, actions[2].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, actions[2].Action, episodeActions[3].Action)

	q = query.GetLastEpisodesActionsQuery{
		UserName: "user1",
		Since:    time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC),
		Limit:    1,
	}
	actions, err = episodesSrv.GetLastActions(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(actions), 1)
	assert.Equal(t, actions[0].Episode, episodeActions[3].URL)
	assert.Equal(t, actions[0].PodcastURL, episodeActions[3].Podcast.URL)
	assert.Equal(t, actions[0].Action, episodeActions[3].Action)
}

func TestEpisodesServiceFavorites(t *testing.T) {
	ctx, i := prepareTests(t)
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
	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: episodeActions})
	assert.NoErr(t, err)

	cmd := command.NewSetFavoriteEpisodeCmd("user1", episodeActions[1].Podcast.URL, episodeActions[1].URL)
	err = settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	cmd = command.NewSetFavoriteEpisodeCmd("user1", episodeActions[3].Podcast.URL, episodeActions[3].URL)
	err = settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	favs, err := episodesSrv.GetFavorites(ctx, "user1")
	assert.NoErr(t, err)
	assert.Equal(t, len(favs), 2)
	assert.Equal(t, favs[0].URL, episodeActions[1].URL)
	assert.Equal(t, favs[1].URL, episodeActions[3].URL)
}

func TestEpisodesServiceNewDevPodcast(t *testing.T) {
	ctx, i := prepareTests(t)
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
		Podcast:   &model.Podcast{URL: "http://example.com/p2"},
		URL:       "http://example.com/p2/ep1",
		Device:    &model.Device{Name: "dev3"}, // new device
		Action:    "delete",
		Timestamp: time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC),
	}

	err := episodesSrv.AddAction(ctx, &command.AddActionCmd{UserName: "user1", Actions: []model.Episode{action}})
	assert.NoErr(t, err)

	q := query.GetEpisodesQuery{
		UserName:   "user1",
		DeviceName: "dev2",
		Aggregated: true,
	}
	episodes, err := episodesSrv.GetEpisodes(ctx, &q)
	assert.NoErr(t, err)
	assert.Equal(t, len(episodes), 1)
	assert.Equal(t, episodes[0].Device.Name, "dev3")
	assert.Equal(t, episodes[0].Podcast.URL, "http://example.com/p2")
}

func prepareEpisodes() []model.Episode {
	var started, position, total int32 = 10, 20, 300

	return []model.Episode{
		{
			Podcast:   &model.Podcast{URL: "http://example.com/p1"},
			URL:       "http://example.com/p1/ep1",
			Device:    &model.Device{Name: "dev1"},
			Action:    "download",
			Timestamp: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		{
			Podcast:   &model.Podcast{URL: "http://example.com/p1"},
			URL:       "http://example.com/p1/ep1",
			Device:    &model.Device{Name: "dev1"},
			Timestamp: time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC),
			Action:    "play",
			Started:   &started,
			Position:  &position,
			Total:     &total,
		},
		{
			Podcast:   &model.Podcast{URL: "http://example.com/p1"},
			URL:       "http://example.com/p1/ep2",
			Device:    &model.Device{Name: "dev1"},
			Action:    "download",
			Timestamp: time.Date(2025, 1, 4, 3, 4, 5, 0, time.UTC),
		},
		{
			Podcast:   &model.Podcast{URL: "http://example.com/p2"},
			URL:       "http://example.com/p2/ep1",
			Device:    &model.Device{Name: "dev2"},
			Action:    "delete",
			Timestamp: time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC),
		},
	}
}

func compareEpisodes(t *testing.T, got, want model.Episode) {
	t.Helper()

	assert.Equal(t, got.URL, want.URL)
	assert.Equal(t, got.Podcast.URL, want.Podcast.URL)
	assert.Equal(t, got.Timestamp, want.Timestamp)
	assert.Equal(t, got.Title, want.Title)

	if want.Device == nil {
		assert.Equal(t, got.Device, nil)

		return
	}

	assert.Equal(t, got.Device.Name, want.Device.Name)
}
