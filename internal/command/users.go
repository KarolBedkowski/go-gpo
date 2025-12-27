package command

// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.

import (
	"errors"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

//---------------------------------------------------------------------

// NewUserCmd define new user to add.
type NewUserCmd struct {
	UserName string
	Password string
	Email    string
	Name     string
}

func (n *NewUserCmd) Validate() error {
	if !validators.IsValidUserName(n.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if n.Password == "" {
		return aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	if n.Email == "" {
		return aerr.ErrValidation.WithUserMsg("email can't be empty")
	}

	return nil
}

// NewUserCmdResult is result of NewUserCmd.
type NewUserCmdResult struct {
	UserID int64
}

//---------------------------------------------------------------------

// ErrChangePasswordOldNotMatch is returned when current user password is other than given.
var ErrChangePasswordOldNotMatch = errors.New("invalid current password")

// ChangeUserPasswordCmd define new user to add.
type ChangeUserPasswordCmd struct {
	UserName         string
	Password         string
	CurrentPassword  string
	CheckCurrentPass bool
}

func (c *ChangeUserPasswordCmd) Validate() error {
	if !validators.IsValidUserName(c.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if c.Password == "" {
		return aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	if c.CheckCurrentPass && c.CurrentPassword == "" {
		return aerr.ErrValidation.WithUserMsg("current password can't be empty")
	}

	return nil
}

//---------------------------------------------------------------------

// LockAccountCmd is user account to lock.
type LockAccountCmd struct {
	UserName string
}

func (l *LockAccountCmd) Validate() error {
	if !validators.IsValidUserName(l.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	return nil
}

//---------------------------------------------------------------------

// DeleteUserCmd delete user and all related data.
type DeleteUserCmd struct {
	UserName string
}

func (d *DeleteUserCmd) Validate() error {
	if !validators.IsValidUserName(d.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	return nil
}
