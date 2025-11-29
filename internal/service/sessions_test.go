package service

//
// sessions_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"testing"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

func TestSessionsService(t *testing.T) {
	_, i := prepareTests(t)
	db := do.MustInvoke[*db.Database](i)
	repo := do.MustInvoke[repository.Sessions](i)

	sessProvider := NewSessionProvider(db, repo, 60*time.Second)

	exists, err := sessProvider.Exist("123")
	assert.NoErr(t, err)
	assert.True(t, !exists)

	store, err := sessProvider.Read("123")
	assert.NoErr(t, err)
	assert.Equal(t, store.ID(), "123")

	exists, err = sessProvider.Exist("123")
	assert.NoErr(t, err)
	assert.True(t, exists)

	assert.NoErr(t, store.Set("abc", 123))
	assert.NoErr(t, store.Set("qwe", "zxc"))

	assert.Equal(t, store.Get("abc"), 123)
	assert.Equal(t, store.Get("qwe"), "zxc")

	assert.NoErr(t, store.Set("qwe", "zxc123"))
	assert.Equal(t, store.Get("qwe"), "zxc123")

	// save to database
	assert.NoErr(t, store.Release())

	// re-read
	store2, err := sessProvider.Read("123")
	assert.NoErr(t, err)
	assert.True(t, store2 != store)
	assert.Equal(t, store2.ID(), "123")

	assert.Equal(t, store.Get("abc"), 123)
	assert.Equal(t, store.Get("qwe"), "zxc123")

	err = store.Delete("abc")
	assert.NoErr(t, err)
	assert.Equal(t, store.Get("abc"), nil)

	// delete all
	assert.NoErr(t, store.Flush())
	assert.Equal(t, len((store.(*SessionStore)).data), 0)
}

func TestSessionsServiceRegenerate(t *testing.T) {
	_, i := prepareTests(t)
	db := do.MustInvoke[*db.Database](i)
	repo := do.MustInvoke[repository.Sessions](i)

	sessProvider := NewSessionProvider(db, repo, 60*time.Second)

	store, err := sessProvider.Read("123")
	assert.NoErr(t, err)
	assert.Equal(t, store.ID(), "123")

	assert.NoErr(t, store.Set("abc", 123))
	assert.NoErr(t, store.Set("qwe", "zxc"))

	// save to database
	assert.NoErr(t, store.Release())

	store2, err := sessProvider.Regenerate("123", "234")
	assert.NoErr(t, err)
	assert.Equal(t, store2.ID(), "234")
	assert.Equal(t, store2.Get("abc"), 123)
	assert.Equal(t, store2.Get("qwe"), "zxc")

	exists, err := sessProvider.Exist("123")
	assert.NoErr(t, err)
	assert.True(t, !exists)

	// re-read
	store3, err := sessProvider.Read("234")
	assert.NoErr(t, err)
	assert.Equal(t, store3.Get("abc"), 123)
	assert.Equal(t, store3.Get("qwe"), "zxc")

	// should be 1 session
	cnt, err := sessProvider.Count()
	assert.NoErr(t, err)
	assert.Equal(t, cnt, 1)
}
