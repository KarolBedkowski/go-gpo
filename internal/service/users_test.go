package service

//
// users_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
)

func TestUsers(t *testing.T) {
	ctx, i := prepareTests(t)
	usersSrv := do.MustInvoke[*UsersSrv](i)

	_, err := usersSrv.LoginUser(ctx, "test", "test123")
	assert.ErrSpec(t, err, common.ErrUnknownUser)

	newuser := command.NewUserCmd{UserName: "test", Password: "test123", Email: "test@example.com", Name: "test user 1"}
	res, err := usersSrv.AddUser(ctx, &newuser)
	assert.NoErr(t, err)
	assert.True(t, res.UserID > 0)

	user, err := usersSrv.LoginUser(ctx, "test", "test123")
	assert.NoErr(t, err)
	assert.Equal(t, user.Name, newuser.Name)
	assert.Equal(t, user.UserName, newuser.UserName)
	assert.Equal(t, user.Email, newuser.Email)

	_, err = usersSrv.LoginUser(ctx, "test", "test1233")
	assert.ErrSpec(t, err, common.ErrUnauthorized)

	// lock account and try login
	err = usersSrv.LockAccount(ctx, command.LockAccountCmd{UserName: "test"})
	assert.NoErr(t, err)
	user, err = usersSrv.LoginUser(ctx, "test", "test123")
	assert.ErrSpec(t, err, common.ErrUserAccountLocked)

	// change pass and unlock
	chpasscmd := command.ChangeUserPasswordCmd{
		UserName:         "test",
		Password:         "123123",
		CurrentPassword:  "",
		CheckCurrentPass: false,
	}
	err = usersSrv.ChangePassword(ctx, &chpasscmd)
	assert.NoErr(t, err)

	user, err = usersSrv.LoginUser(ctx, "test", "123123")
	assert.NoErr(t, err)

	// try double user
	newuser2 := command.NewUserCmd{
		UserName: "test",
		Password: "test123",
		Email:    "test2@example.com",
		Name:     "test user 2",
	}
	_, err = usersSrv.AddUser(ctx, &newuser2)
	assert.ErrSpec(t, err, common.ErrUserExists)

	newuser2.UserName = "test2"
	res2, err := usersSrv.AddUser(ctx, &newuser2)
	assert.NoErr(t, err)
	assert.True(t, res2.UserID > 0)
	assert.True(t, res.UserID != res2.UserID)

	// get all users
	users, err := usersSrv.GetUsers(ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 2)
	assert.Equal(t, users[0].UserName, "test")
	assert.Equal(t, users[1].UserName, "test2")

	// lock test2
	err = usersSrv.LockAccount(ctx, command.LockAccountCmd{UserName: "test2"})
	assert.NoErr(t, err)

	// get active users
	users, err = usersSrv.GetUsers(ctx, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 1)
	assert.Equal(t, users[0].UserName, "test")
}

func TestDeleteUser(t *testing.T) {
	ctx, i := prepareTests(t)
	usersSrv := do.MustInvoke[*UsersSrv](i)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
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

	err := usersSrv.DeleteUser(ctx, &command.DeleteUserCmd{UserName: "user3"})
	assert.ErrSpec(t, err, common.ErrUnknownUser)

	err = usersSrv.DeleteUser(ctx, &command.DeleteUserCmd{UserName: "user2"})
	assert.NoErr(t, err)

	// get active users
	users, err := usersSrv.GetUsers(ctx, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 1)
	assert.Equal(t, users[0].UserName, "user1")

	err = usersSrv.DeleteUser(ctx, &command.DeleteUserCmd{UserName: "user1"})
	assert.NoErr(t, err)

	// get active users
	users, err = usersSrv.GetUsers(ctx, true)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 0)
}
