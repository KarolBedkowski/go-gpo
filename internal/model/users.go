// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "time"

type User struct {
	ID        int
	Username  string
	Password  string
	Email     string
	Name      string
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (u *User) CheckPassword(pass string) bool {
	// TODO: hash
	return pass == u.Password
}
