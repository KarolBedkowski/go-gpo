//
// users.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrUserExists   = errors.New("user already exists")
)

type Users struct {
	db         *db.Database
	passHasher PasswordHasher
}

func NewUsersService(db *db.Database) *Users {
	return &Users{db, BCryptPasswordHasher{}}
}

func (u *Users) LoginUser(ctx context.Context, username, password string) (model.User, error) {
	conn, err := u.db.GetConnection(ctx)
	if err != nil {
		return model.User{}, fmt.Errorf("get db connection error: %w", err)
	}

	defer conn.Close()

	repo := u.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return model.User{}, ErrUnknownUser
	} else if err != nil {
		return model.User{}, fmt.Errorf("get user error: %w", err)
	}

	if !u.passHasher.CheckPassword(password, user.Password) {
		return model.User{}, ErrUnauthorized
	}

	return model.NewUserFromUserDB(&user), nil
}

func (u *Users) AddUser(ctx context.Context, user model.User) (int64, error) {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("get db connection error: %w", err)
	}

	defer tx.Rollback()

	repo := u.db.GetRepository(tx)

	// is user exists?
	if _, err := repo.GetUser(ctx, user.Username); err != nil && !errors.Is(err, repository.ErrNoData) {
		return 0, fmt.Errorf("check user exists error: %w", err)
	} else if err == nil {
		return 0, ErrUserExists
	}

	hashedPass, err := u.passHasher.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("hash password error: %w", err)
	}

	now := time.Now()
	udb := repository.UserDB{
		Username:  user.Username,
		Password:  hashedPass,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	id, err := repo.SaveUser(ctx, &udb)
	if err != nil {
		return 0, fmt.Errorf("save user error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit error: %w", err)
	}

	return id, nil
}

func (u *Users) ChangePassword(ctx context.Context, user model.User) (int64, error) {
	tx, err := u.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Rollback()

	repo := u.db.GetRepository(tx)

	// is user exists?
	udb, err := repo.GetUser(ctx, user.Username)
	if errors.Is(err, repository.ErrNoData) {
		return 0, ErrUnknownUser
	} else if err != nil {
		return 0, fmt.Errorf("get user error: %w", err)
	}

	udb.Password, err = u.passHasher.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("hash password error: %w", err)
	}

	udb.UpdatedAt = time.Now()

	id, err := repo.SaveUser(ctx, &udb)
	if err != nil {
		return 0, fmt.Errorf("save user error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit error: %w", err)
	}

	return id, nil
}

//---------------------------

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	CheckPassword(password, hash string) bool
}

type BCryptPasswordHasher struct{}

func (BCryptPasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(hash), err
}

func (BCryptPasswordHasher) CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
