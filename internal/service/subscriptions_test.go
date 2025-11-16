package service

//
// subs_test.go
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
	"gitlab.com/kabes/go-gpo/internal/queries"
)

func TestSubsServiceUser(t *testing.T) {
	ctx, i := prepareTests(t)
	subsSrv := do.MustInvoke[*SubscriptionsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev1",
		Subscriptions: newSubscribed,
		Timestamp:     time.Now().UTC(),
	}
	err := subsSrv.ReplaceSubscriptions(ctx, &cmd)
	assert.NoErr(t, err)

	// getsubs
	subs, err := subsSrv.GetUserSubscriptions(ctx, "user1", time.Time{})
	assert.Equal(t, subs, newSubscribed)

	// replace
	newSubscribed2 := []string{
		"http://example.com/p1",
		"http://example.com/p4",
		"http://example.com/p5",
	}

	cmd2 := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev1",
		Subscriptions: newSubscribed2,
		Timestamp:     time.Now().UTC(),
	}
	err = subsSrv.ReplaceSubscriptions(ctx, &cmd2)
	assert.NoErr(t, err)

	// getsubs
	subs, err = subsSrv.GetUserSubscriptions(ctx, "user1", time.Time{})
	assert.Equal(t, subs, newSubscribed2)
}

func TestSubsServiceDevice(t *testing.T) {
	ctx, i := prepareTests(t)
	subsSrv := do.MustInvoke[*SubscriptionsSrv](i)
	deviceSrv := do.MustInvoke[*DevicesSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev1",
		Subscriptions: newSubscribed,
		Timestamp:     time.Now().UTC(),
	}
	err := subsSrv.ReplaceSubscriptions(ctx, &cmd)
	assert.NoErr(t, err)

	// replace with other device
	newSubscribed2 := []string{
		"http://example.com/p1",
		"http://example.com/p4",
		"http://example.com/p5",
	}

	// new device - should be created
	cmd2 := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev2",
		Subscriptions: newSubscribed2,
		Timestamp:     time.Now().UTC(),
	}
	err = subsSrv.ReplaceSubscriptions(ctx, &cmd2)
	assert.NoErr(t, err)

	devices, err := deviceSrv.ListDevices(ctx, &queries.QueryDevices{UserName: "user1"})
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)

	// getsubs
	subs, err := subsSrv.GetUserSubscriptions(ctx, "user1", time.Time{})
	assert.Equal(t, subs, newSubscribed2)

	// all devices should have the same subscriptions list
	subs, err = subsSrv.GetSubscriptions(ctx, "user1", "dev1", time.Time{})
	assert.NoErr(t, err)
	assert.Equal(t, subs, newSubscribed2)

	subs, err = subsSrv.GetSubscriptions(ctx, "user1", "dev2", time.Time{})
	assert.NoErr(t, err)
	assert.Equal(t, subs, newSubscribed2)
}

func TestSubsServiceChanges(t *testing.T) {
	ctx, i := prepareTests(t)
	subsSrv := do.MustInvoke[*SubscriptionsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev1",
		Subscriptions: newSubscribed,
		Timestamp:     time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	err := subsSrv.ReplaceSubscriptions(ctx, &cmd)
	assert.NoErr(t, err)

	state, err := subsSrv.GetSubscriptionChanges(ctx, "user1", "dev1", time.Time{})
	assert.NoErr(t, err)
	assert.Equal(t, len(state.Removed), 0)
	assert.EqualSorted(t, state.AddedURLs(), newSubscribed)

	// no new
	state, err = subsSrv.GetSubscriptionChanges(ctx, "user1", "dev1",
		time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC))
	assert.NoErr(t, err)
	assert.Equal(t, len(state.Removed), 0)
	assert.Equal(t, len(state.Added), 0)

	// replace with other device
	newSubscribed2 := []string{
		"http://example.com/p1",
		"http://example.com/p4",
		"http://example.com/p5",
	}

	// new device - should be created
	cmd2 := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev2",
		Subscriptions: newSubscribed2,
		Timestamp:     time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC),
	}
	err = subsSrv.ReplaceSubscriptions(ctx, &cmd2)
	assert.NoErr(t, err)

	// new
	state, err = subsSrv.GetSubscriptionChanges(ctx, "user1", "dev1",
		time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC))
	assert.NoErr(t, err)
	assert.EqualSorted(t, state.RemovedURLs(), []string{"http://example.com/p2", "http://example.com/p3"})
	assert.EqualSorted(t, state.AddedURLs(), []string{"http://example.com/p4", "http://example.com/p5"})

	// no new at 12:01
	state, err = subsSrv.GetSubscriptionChanges(ctx, "user1", "dev1",
		time.Date(2025, 1, 2, 12, 1, 0, 0, time.UTC))
	assert.NoErr(t, err)
	assert.Equal(t, len(state.Removed), 0)
	assert.Equal(t, len(state.Added), 0)
}

func TestSubsServiceUpdateDevSubsChanges(t *testing.T) {
	ctx, i := prepareTests(t)
	subsSrv := do.MustInvoke[*SubscriptionsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")

	// init some data
	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      "user1",
		DeviceName:    "dev1",
		Subscriptions: newSubscribed,
		Timestamp:     time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	err := subsSrv.ReplaceSubscriptions(ctx, &cmd)
	assert.NoErr(t, err)

	changes := command.ChangeSubscriptionsCmd{
		UserName:   "user1",
		DeviceName: "dev1",
		Add:        []string{"http://example.com/p4", "http://example.com/p5"},
		Remove:     []string{"http://example.com/p1"},
		Timestamp:  time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC),
	}
	assert.NoErr(t, changes.Validate())

	_, err = subsSrv.ChangeSubscriptions(ctx, &changes)
	assert.NoErr(t, err)

	// new
	state, err := subsSrv.GetSubscriptionChanges(ctx, "user1", "dev1",
		time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC))
	assert.NoErr(t, err)
	assert.EqualSorted(t, state.RemovedURLs(), []string{"http://example.com/p1"})
	assert.EqualSorted(t, state.AddedURLs(), []string{"http://example.com/p4", "http://example.com/p5"})

	// check for other device; should be the same
	state, err = subsSrv.GetSubscriptionChanges(ctx, "user1", "dev2",
		time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC))
	assert.NoErr(t, err)
	assert.EqualSorted(t, state.RemovedURLs(), []string{"http://example.com/p1"})
	assert.EqualSorted(t, state.AddedURLs(), []string{"http://example.com/p4", "http://example.com/p5"})
}
