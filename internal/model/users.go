// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "github.com/rs/zerolog"

const UserLockedPassword = "LOCKED"

type User struct {
	ID       int64
	UserName string
	Password string
	Email    string
	Name     string

	Locked bool
}

func (u *User) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", u.ID).
		Str("user_name", u.UserName).
		Str("email", u.Email).
		Str("name", u.Name).
		Bool("locked", u.Locked)
}
