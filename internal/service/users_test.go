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

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestUsers(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	usersSrv := do.MustInvoke[*Users](i)

	_, err := usersSrv.LoginUser(ctx, "test", "test123")
	assert.ErrSpec(t, err, ErrUnknownUser)

	newuser, _ := model.NewNewUser("test", "test123", "test@example.com", "test user 1")

	uid, err := usersSrv.AddUser(ctx, &newuser)
	assert.NoErr(t, err)
	assert.True(t, uid > 0)

	user, err := usersSrv.LoginUser(ctx, "test", "test123")
	assert.NoErr(t, err)
	assert.Equal(t, user.Name, newuser.Name)
	assert.Equal(t, user.Username, newuser.Username)
	assert.Equal(t, user.Email, newuser.Email)

	_, err = usersSrv.LoginUser(ctx, "test", "test1233")
	assert.ErrSpec(t, err, ErrUnauthorized)

	// lock account and try login
	err = usersSrv.LockAccount(ctx, model.LockAccount{Username: "test"})
	assert.NoErr(t, err)
	user, err = usersSrv.LoginUser(ctx, "test", "test123")
	assert.ErrSpec(t, err, ErrUserAccountLocked)

	// change pass and unlock
	err = usersSrv.ChangePassword(ctx, &model.UserPassword{Username: "test", Password: "123123"})
	assert.NoErr(t, err)

	user, err = usersSrv.LoginUser(ctx, "test", "123123")
	assert.NoErr(t, err)

	// try double user
	newuser2, _ := model.NewNewUser("test", "test123", "test2@example.com", "test user 2")
	_, err = usersSrv.AddUser(ctx, &newuser2)
	assert.ErrSpec(t, err, ErrUserExists)

	newuser2.Username = "test2"
	uid2, err := usersSrv.AddUser(ctx, &newuser2)
	assert.NoErr(t, err)
	assert.True(t, uid2 > 0)
	assert.True(t, uid != uid2)

	// get all users
	users, err := usersSrv.GetUsers(ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 2)
	assert.Equal(t, users[0].Username, "test")
	assert.Equal(t, users[1].Username, "test2")

	// lock test2
	err = usersSrv.LockAccount(ctx, model.LockAccount{Username: "test2"})
	assert.NoErr(t, err)

	// get active users
	users, err = usersSrv.GetUsers(ctx, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 1)
	assert.Equal(t, users[0].Username, "test")
}
