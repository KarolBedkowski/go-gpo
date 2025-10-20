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

	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrUserExists   = errors.New("user already exists")
)

type Users struct {
	repo       *repository.Repository
	passHasher PasswordHasher
}

func NewUsersService(repo *repository.Repository) *Users {
	return &Users{repo, BCryptPasswordHasher{}}
}

func (u *Users) LoginUser(ctx context.Context, username, password string) (*model.User, error) {
	user, err := u.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	if user == nil {
		return nil, ErrUnknownUser
	}

	if !u.passHasher.CheckPassword(password, user.Password) {
		return nil, ErrUnauthorized
	}

	return model.NewUserFromUserDB(user), nil
}

func (u *Users) AddUser(ctx context.Context, user *model.User) (int64, error) {
	// is user exists?
	if eu, err := u.repo.GetUser(ctx, user.Username); err != nil {
		return 0, fmt.Errorf("check user exists error: %w", err)
	} else if eu != nil {
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

	id, err := u.repo.SaveUser(ctx, &udb)
	if err != nil {
		return 0, fmt.Errorf("save user error: %w", err)
	}

	return id, nil
}

func (u *Users) ChangePassword(ctx context.Context, user *model.User) (int64, error) {
	// is user exists?
	udb, err := u.repo.GetUser(ctx, user.Username)
	if err != nil {
		return 0, fmt.Errorf("get user error: %w", err)
	} else if udb == nil {
		return 0, ErrUnknownUser
	}

	udb.Password, err = u.passHasher.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("hash password error: %w", err)
	}

	udb.UpdatedAt = time.Now()

	id, err := u.repo.SaveUser(ctx, udb)
	if err != nil {
		return 0, fmt.Errorf("save user error: %w", err)
	}

	return id, nil
}

//-----------------

const CtxUserKey = "CtxUserKey"

func ContextUser(ctx context.Context) string {
	suser, ok := ctx.Value(CtxUserKey).(string)
	if ok {
		return suser
	}

	return ""
}

func ContextWithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, CtxUserKey, username)
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
