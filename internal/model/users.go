// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "github.com/rs/zerolog"

const UserLockedPassword = "LOCKED"

type User struct {
	UserName string
	Password string
	Email    string
	Name     string
	ID       int32

	Locked bool
}

func (u *User) MarshalZerologObject(event *zerolog.Event) {
	event.Int32("id", u.ID).
		Str("user_name", u.UserName).
		Str("email", u.Email).
		Str("name", u.Name).
		Bool("locked", u.Locked)
}
