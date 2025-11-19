package service

//
// mod_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"context"
	stdlog "log"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func prepareTests(t *testing.T) (context.Context, *do.RootScope) {
	t.Helper()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Caller().Stack().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	ctx := log.Logger.WithContext(context.Background())
	i := do.New(Package, db.Package, repository.Package)

	db := do.MustInvoke[*db.Database](i)
	if err := db.Connect(ctx, "sqlite3", ":memory:"); err != nil {
		t.Fatalf("connect to db error: %#+v", err)
	}

	if err := db.Migrate(ctx, "sqlite3"); err != nil {
		t.Fatalf("prepare db error: %#+v", err)
	}

	return ctx, i
}

func prepareTestUser(ctx context.Context, t *testing.T, i do.Injector, name string) int64 {
	t.Helper()

	newuser := command.NewUserCmd{
		UserName: name,
		Password: name + "123",
		Email:    name + "@example.com",
		Name:     "test user " + name,
	}
	usersSrv := do.MustInvoke[*UsersSrv](i)
	res, err := usersSrv.AddUser(ctx, &newuser)
	if err != nil {
		t.Fatalf("create test user failed: %#+v", err)
	}

	return res.UserID
}

func prepareTestDevice(ctx context.Context, t *testing.T, i do.Injector,
	username, devicename string,
) {
	t.Helper()

	deviceSrv := do.MustInvoke[*DevicesSrv](i)
	cmd := command.UpdateDeviceCmd{
		UserName:   username,
		DeviceName: devicename,
		DeviceType: "other",
		Caption:    "device " + devicename + " caption",
	}
	err := deviceSrv.UpdateDevice(ctx, &cmd)
	if err != nil {
		t.Fatalf("create test device failed: %#+v", err)
	}
}

func prepareTestSub(ctx context.Context, t *testing.T, i do.Injector,
	username, devicename string, subs ...string,
) {
	t.Helper()

	subsSrv := do.MustInvoke[*SubscriptionsSrv](i)
	cmd := command.ReplaceSubscriptionsCmd{
		UserName:      username,
		DeviceName:    devicename,
		Subscriptions: subs,
		Timestamp:     time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	err := subsSrv.ReplaceSubscriptions(ctx, &cmd)
	assert.NoErr(t, err)
}

func prepareTestEpisode(ctx context.Context, t *testing.T, i do.Injector,
	username, devicename, podcast string, episode ...string,
) {
	t.Helper()

	episodesSrv := do.MustInvoke[*EpisodesSrv](i)

	for _, ep := range episode {
		action := model.Episode{
			Podcast:   podcast,
			Episode:   ep,
			Device:    devicename,
			Action:    "download",
			Timestamp: time.Date(2025, 1, 5, 3, 4, 5, 0, time.UTC),
		}

		err := episodesSrv.AddAction(ctx, &command.AddActionCmd{
			UserName: username, Actions: []model.Episode{action},
		})
		assert.NoErr(t, err)
	}
}
