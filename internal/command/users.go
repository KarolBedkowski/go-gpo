// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package command

import (
	"errors"

	"gitlab.com/kabes/go-gpo/internal/aerr"
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
	if n.UserName == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	if n.Password == "" {
		return aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	return nil
}

type NewUserCmdResult struct {
	UserID int64
}

//---------------------------------------------------------------------

var ErrChangePasswordOldNotMatch = errors.New("invalid current password")

// ChangeUserPasswordCmd define new user to add.
type ChangeUserPasswordCmd struct {
	UserName         string
	Password         string
	CurrentPassword  string
	CheckCurrentPass bool
}

func (c *ChangeUserPasswordCmd) Validate() error {
	if c.UserName == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
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
	if l.UserName == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return nil
}

//---------------------------------------------------------------------

// DeleteUserCmd delete user and all related data.
type DeleteUserCmd struct {
	UserName string
}

func (d *DeleteUserCmd) Validate() error {
	if d.UserName == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return nil
}
