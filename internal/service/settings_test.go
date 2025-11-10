package service

//
// users_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"context"
	"testing"

	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestSettingsAccount(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	settSrv := do.MustInvoke[*Settings](i)
	_ = prepareTestUser(ctx, t, i, "test1")
	_ = prepareTestUser(ctx, t, i, "test2")

	u1skey, err := model.NewSettingsKey("test1", "account", "", "", "")
	assert.NoErr(t, err)
	u1set1 := map[string]string{"key1": "val1", "key2": "val2"}

	err = settSrv.SaveSettings(ctx, &u1skey, u1set1)
	assert.NoErr(t, err)

	rset, err := settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, u1set1)

	u1set2 := map[string]string{"key1": "val1-new", "key3": "val3"}
	err = settSrv.SaveSettings(ctx, &u1skey, u1set2)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	// add settings for other user
	u2skey, err := model.NewSettingsKey("test2", "account", "", "", "")
	set2 := map[string]string{"key1": "u2val1", "key3": "u2val3"}
	err = settSrv.SaveSettings(ctx, &u2skey, set2)
	assert.NoErr(t, err)

	// check
	rset, err = settSrv.GetSettings(ctx, &u2skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset, set2)

	// check first user
	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 3)
	assert.Equal(t, rset["key1"], "val1-new")
	assert.Equal(t, rset["key2"], "val2")
	assert.Equal(t, rset["key3"], "val3")

	// delete setting
	u1set3 := map[string]string{"key1": "val2-new", "key3": ""}
	err = settSrv.SaveSettings(ctx, &u1skey, u1set3)
	assert.NoErr(t, err)

	rset, err = settSrv.GetSettings(ctx, &u1skey)
	assert.NoErr(t, err)
	assert.Equal(t, len(rset), 2)
	assert.Equal(t, rset["key1"], "val2-new")
	assert.Equal(t, rset["key2"], "val2")
}
