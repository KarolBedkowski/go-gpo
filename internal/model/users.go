// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"strings"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

const UserLockedPassword = "LOCKED"

type User struct {
	Username string
	Password string
	Email    string
	Name     string

	Locked bool
}

func NewUserFromUserDB(u *repository.UserDB) User {
	return User{
		Username: u.Username,
		Password: u.Password,
		Email:    u.Email,
		Name:     u.Name,
		Locked:   u.Password == UserLockedPassword,
	}
}

//---------------------------------------------------------------------

// NewUser define new user to add.
type NewUser struct {
	Username string
	Password string
	Email    string
	Name     string
}

func NewNewUser(username, password, email, name string) NewUser {
	return NewUser{
		Name:     strings.TrimSpace(name),
		Password: strings.TrimSpace(password),
		Email:    strings.TrimSpace(email),
		Username: strings.TrimSpace(username),
	}
}

func (n *NewUser) Validate() error {
	if n.Username == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	if n.Password == "" {
		return aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	return nil
}

//---------------------------------------------------------------------

// NewUser define new user to add.
type UserPassword struct {
	Username string
	Password string
}

func NewUserPassword(username, password string) UserPassword {
	return UserPassword{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
	}
}

func (u *UserPassword) Validate() error {
	if u.Username == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	if u.Password == "" {
		return aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	return nil
}

//---------------------------------------------------------------------

// LockAccount is user account to lock.
type LockAccount struct {
	Username string
}

func NewLockAccount(username string) LockAccount {
	return LockAccount{
		Username: strings.TrimSpace(username),
	}
}

func (l *LockAccount) Validate() error {
	if l.Username == "" {
		return aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return nil
}
