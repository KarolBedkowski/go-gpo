// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package command

import "errors"

//---------------------------------------------------------------------

// NewUserCmd define new user to add.
type NewUserCmd struct {
	Username string
	Password string
	Email    string
	Name     string
}

type NewUserCmdResult struct {
	Success bool
	UserID  int64
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

type ChangeUserPasswordCmdResult struct {
	Success bool
}

//---------------------------------------------------------------------

// LockAccountCmd is user account to lock.
type LockAccountCmd struct {
	Username string
}

type LockAccountCmdResult struct {
	Success bool
}
