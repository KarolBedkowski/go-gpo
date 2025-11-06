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

func NewNewUser(username, password, email, name string) (NewUser, error) {
	user := NewUser{
		Name:     strings.TrimSpace(name),
		Password: strings.TrimSpace(password),
		Email:    strings.TrimSpace(email),
		Username: strings.TrimSpace(username),
	}

	if user.Username == "" {
		return NewUser{}, aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	if user.Password == "" {
		return NewUser{}, aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	return user, nil
}

//---------------------------------------------------------------------

// NewUser define new user to add.
type UserPassword struct {
	Username string
	Password string
}

func NewUserPassword(username, password string) (UserPassword, error) {
	userpass := UserPassword{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
	}

	if userpass.Username == "" {
		return UserPassword{}, aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	if userpass.Password == "" {
		return UserPassword{}, aerr.ErrValidation.WithUserMsg("password can't be empty")
	}

	return userpass, nil
}

//---------------------------------------------------------------------

// LockAccount is user account to lock.
type LockAccount struct {
	Username string
}

func NewLockAccount(username string) (LockAccount, error) {
	la := LockAccount{
		Username: strings.TrimSpace(username),
	}

	if la.Username == "" {
		return LockAccount{}, aerr.ErrValidation.WithUserMsg("username can't be empty")
	}

	return la, nil
}
