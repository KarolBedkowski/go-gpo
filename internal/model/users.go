// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "time"

type User struct {
	Username string
	Password string
	Email    string
	Name     string
}

func NewUserFromUserDB(u *UserDB) *User {
	return &User{
		Username: u.Username,
		Password: u.Password,
		Email:    u.Email,
		Name:     u.Name,
	}
}

type UserDB struct {
	ID        int       `db:"id"`
	Username  string    `db:"username"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (u *UserDB) CheckPassword(pass string) bool {
	// TODO: hash
	return pass == u.Password
}
