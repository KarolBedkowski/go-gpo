// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"gitlab.com/kabes/go-gpo/internal/repository"
)

const UserLockedPassword = "LOCKED"

type User struct {
	UserName string
	Password string
	Email    string
	Name     string

	Locked bool
}

func NewUserFromUserDB(u *repository.UserDB) User {
	return User{
		UserName: u.UserName,
		Password: u.Password,
		Email:    u.Email,
		Name:     u.Name,
		Locked:   u.Password == UserLockedPassword,
	}
}
