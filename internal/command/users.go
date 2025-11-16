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
	Username string
	Password string
	Email    string
	Name     string
}

func (n *NewUserCmd) Validate() error {
	if n.Username == "" {
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
	Username         string
	Password         string
	CurrentPassword  string
	CheckCurrentPass bool
}

func (c *ChangeUserPasswordCmd) Validate() error {
	if c.Username == "" {
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
	Username string
}

func (l *LockAccountCmd) Validate() error {
	if l.Username == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return nil
}

//---------------------------------------------------------------------

// DeleteUserCmd delete user and all related data.
type DeleteUserCmd struct {
	Username string
}

func (d *DeleteUserCmd) Validate() error {
	if d.Username == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return nil
}
