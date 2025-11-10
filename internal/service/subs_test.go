package service

//
// subs_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"testing"
	"time"

	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestSubsServiceUser(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	subsSrv := do.MustInvoke[*Subs](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	_ = subsSrv

	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	err := subsSrv.UpdateDeviceSubscriptions(ctx, "user1", "dev1", model.NewSubscribedURLS(newSubscribed), time.Now())
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

	err = subsSrv.UpdateDeviceSubscriptions(ctx, "user1", "dev1", model.NewSubscribedURLS(newSubscribed2), time.Now())
	assert.NoErr(t, err)

	// getsubs
	subs, err = subsSrv.GetUserSubscriptions(ctx, "user1", time.Time{})
	assert.Equal(t, subs, newSubscribed2)
}

func TestSubsServiceDevice(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	subsSrv := do.MustInvoke[*Subs](i)
	deviceSrv := do.MustInvoke[*Device](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")

	newSubscribed := []string{
		"http://example.com/p1",
		"http://example.com/p2",
		"http://example.com/p3",
	}

	err := subsSrv.UpdateDeviceSubscriptions(ctx, "user1", "dev1", model.NewSubscribedURLS(newSubscribed), time.Now())
	assert.NoErr(t, err)

	// replace with other device
	newSubscribed2 := []string{
		"http://example.com/p1",
		"http://example.com/p4",
		"http://example.com/p5",
	}

	// new device - should be created
	err = subsSrv.UpdateDeviceSubscriptions(ctx, "user1", "dev2", model.NewSubscribedURLS(newSubscribed2), time.Now())
	assert.NoErr(t, err)

	devices, err := deviceSrv.ListDevices(ctx, "user1")
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)

	// getsubs
	subs, err := subsSrv.GetUserSubscriptions(ctx, "user1", time.Time{})
	assert.Equal(t, subs, newSubscribed2)

	// all devices should have the same subscriptions list
	subs, err = subsSrv.GetDeviceSubscriptions(ctx, "user1", "dev1", time.Time{})
	assert.NoErr(t, err)
	assert.Equal(t, subs, newSubscribed2)

	subs, err = subsSrv.GetDeviceSubscriptions(ctx, "user1", "dev2", time.Time{})
	assert.NoErr(t, err)
	assert.Equal(t, subs, newSubscribed2)
}
