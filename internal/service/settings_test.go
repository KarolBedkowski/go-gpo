package service

//
// users_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"testing"

	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/query"
)

func TestSettingsAccount(t *testing.T) {
	ctx, i := prepareTests(t)
	settSrv := do.MustInvoke[*SettingsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")

	cmd := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "account",
		Set:      map[string]string{"key1": "val1", "key2": "val2"},
	}
	err := settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	u1skey := query.SettingsQuery{UserName: "user1", Scope: "account"}
	rset, err := settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, cmd.Set)

	cmd2 := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "account",
		Set:      map[string]string{"key1": "val1-new", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd2)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	// add settings for other user
	cmd3 := command.ChangeSettingsCmd{
		UserName: "user2",
		Scope:    "account",
		Set:      map[string]string{"key1": "u2val1", "key3": "u2val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd3)
	assert.NoErr(t, err)

	// check
	u2skey := query.SettingsQuery{UserName: "user2", Scope: "account"}
	rset, err = settSrv.GetSettings(ctx, &u2skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, cmd3.Set)

	// check first user
	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	// delete setting
	cmd4 := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "account",
		Set:      map[string]string{"key1": "val2-new"},
		Remove:   []string{"key3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd4)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset["key1"], "val2-new")
	assert.Equal(t, rset["key2"], "val2")
}

func TestSettingsDevice(t *testing.T) {
	ctx, i := prepareTests(t)
	settSrv := do.MustInvoke[*SettingsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")

	cmd := command.ChangeSettingsCmd{
		UserName:   "user1",
		Scope:      "device",
		DeviceName: "dev1",
		Set:        map[string]string{"key1": "val1", "key2": "val2"},
	}
	err := settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	cmd2 := command.ChangeSettingsCmd{
		UserName:   "user1",
		Scope:      "device",
		DeviceName: "dev2",
		Set:        map[string]string{"key1": "val1-d2", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd2)
	assert.NoErr(t, err)

	d1skey := query.SettingsQuery{UserName: "user1", Scope: "device", DeviceName: "dev1", Podcast: "", Episode: ""}
	rset, err := settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, cmd.Set)

	cmd3 := command.ChangeSettingsCmd{
		UserName:   "user1",
		Scope:      "device",
		DeviceName: "dev1",
		Set:        map[string]string{"key1": "val1-new", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd3)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	d2skey := query.SettingsQuery{UserName: "user1", Scope: "device", DeviceName: "dev2", Podcast: "", Episode: ""}
	rset, err = settSrv.GetSettings(ctx, &d2skey)
	assert.NoErr(t, err)
	assert.Equal(t, rset, cmd2.Set)
}

func TestSettingsPdocast(t *testing.T) {
	ctx, i := prepareTests(t)
	settSrv := do.MustInvoke[*SettingsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
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

	cmd := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "podcast",
		Podcast:  "http://example.com/p1",
		Set:      map[string]string{"key1": "val1", "key2": "val2"},
	}
	err := settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	cmd2 := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "podcast",
		Podcast:  "http://example.com/p2",
		Set:      map[string]string{"key1": "val1-d2", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd2)
	assert.NoErr(t, err)

	d1skey := query.SettingsQuery{
		UserName:   "user1",
		Scope:      "podcast",
		DeviceName: "dev1",
		Podcast:    "http://example.com/p1",
		Episode:    "",
	}
	rset, err := settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, cmd.Set)

	cmd3 := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "podcast",
		Podcast:  "http://example.com/p1",
		Set:      map[string]string{"key1": "val1-new", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd3)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	d2skey := query.SettingsQuery{
		UserName:   "user1",
		Scope:      "podcast",
		DeviceName: "dev2",
		Podcast:    "http://example.com/p2",
		Episode:    "",
	}
	rset, err = settSrv.GetSettings(ctx, &d2skey)
	assert.NoErr(t, err)
	assert.Equal(t, rset, cmd2.Set)
}

func TestSettingsepisode(t *testing.T) {
	ctx, i := prepareTests(t)
	settSrv := do.MustInvoke[*SettingsSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
	prepareTestSub(
		ctx, t, i, "user1", "dev1", "http://example.com/p1", "http://example.com/p2",
	)
	prepareTestEpisode(ctx, t, i,
		"user1", "dev1", "http://example.com/p1", "http://example.com/p1/e1", "http://example.com/p1/e2",
	)
	prepareTestEpisode(ctx, t, i,
		"user1", "dev1", "http://example.com/p2", "http://example.com/p2/e1", "http://example.com/p2/e2",
	)

	cmd := command.ChangeSettingsCmd{
		UserName:   "user1",
		Scope:      "episode",
		DeviceName: "dev1", // should be ignored
		Podcast:    "http://example.com/p1",
		Episode:    "http://example.com/p1/e1",
		Set:        map[string]string{"key1": "val1", "key2": "val2"},
	}
	err := settSrv.SaveSettings(ctx, &cmd)
	assert.NoErr(t, err)

	cmd2 := command.ChangeSettingsCmd{
		UserName:   "user1",
		Scope:      "episode",
		DeviceName: "dev2", // should be ignored
		Podcast:    "http://example.com/p2",
		Episode:    "http://example.com/p2/e2",
		Set:        map[string]string{"key1": "val1-d2", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd2)
	assert.NoErr(t, err)

	d1skey := query.SettingsQuery{
		UserName:   "user1",
		Scope:      "episode",
		DeviceName: "dev1",
		Podcast:    "http://example.com/p1",
		Episode:    "http://example.com/p1/e1",
	}
	rset, err := settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, cmd.Set)

	cmd3 := command.ChangeSettingsCmd{
		UserName: "user1",
		Scope:    "episode",
		Podcast:  "http://example.com/p1",
		Episode:  "http://example.com/p1/e1",
		Set:      map[string]string{"key1": "val1-new", "key3": "val3"},
	}
	err = settSrv.SaveSettings(ctx, &cmd3)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &d1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	d2skey := query.SettingsQuery{
		UserName:   "user1",
		Scope:      "episode",
		DeviceName: "dev2",
		Podcast:    "http://example.com/p2",
		Episode:    "http://example.com/p2/e2",
	}
	rset, err = settSrv.GetSettings(ctx, &d2skey)
	assert.NoErr(t, err)
	assert.Equal(t, rset, cmd2.Set)
}
