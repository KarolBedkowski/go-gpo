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
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized      = aerr.New("unauthorized").WithUserMsg("authorization failed")
	ErrUserAccountLocked = aerr.New("locked account").WithUserMsg("account is locked")
	ErrUserExists        = aerr.New("username exists").WithUserMsg("user name already exists")
)

type Users struct {
	db         *db.Database
	usersRepo  repository.UsersRepository
	passHasher PasswordHasher
}

func NewUsersServiceI(i do.Injector) (*Users, error) {
	db := do.MustInvoke[*db.Database](i)
	repo := do.MustInvoke[repository.UsersRepository](i)

	return &Users{db, repo, BCryptPasswordHasher{}}, nil
}

func (u *Users) LoginUser(ctx context.Context, username, password string) (model.User, error) {
	conn, err := u.db.GetConnection(ctx)
	if err != nil {
		return model.User{}, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := u.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return model.User{}, ErrUnknownUser
	} else if err != nil {
		return model.User{}, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if user.Password == model.UserLockedPassword {
		return model.User{}, ErrUserAccountLocked
	}

	if !u.passHasher.CheckPassword(password, user.Password) {
		return model.User{}, ErrUnauthorized
	}

	return model.NewUserFromUserDB(&user), nil
}

func (u *Users) AddUser(ctx context.Context, user model.NewUser) (int64, error) {
	//nolint:wrapcheck
	return db.InTransactionR(ctx, u.db, func(dbctx repository.DBContext) (int64, error) {
		// is user exists?
		_, err := u.usersRepo.GetUser(ctx, dbctx, user.Username)
		switch {
		case errors.Is(err, repository.ErrNoData):
			// ok; user not exists
		case err == nil:
			// user exists
			return 0, ErrUserExists
		default:
			// failed to get user
			return 0, aerr.ApplyFor(ErrRepositoryError, err)
		}

		hashedPass, err := u.passHasher.HashPassword(user.Password)
		if err != nil {
			return 0, aerr.Wrapf(err, "hash password failed")
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

		return u.usersRepo.SaveUser(ctx, dbctx, &udb)
	})
}

func (u *Users) ChangePassword(ctx context.Context, user model.UserPassword) error {
	//nolint: wrapcheck
	return u.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		// is user exists?
		udb, err := u.usersRepo.GetUser(ctx, dbctx, user.Username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		udb.Password, err = u.passHasher.HashPassword(user.Password)
		if err != nil {
			return aerr.Wrapf(err, "hash password failed")
		}

		udb.UpdatedAt = time.Now()

		if _, err = u.usersRepo.SaveUser(ctx, dbctx, &udb); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})
}

func (u *Users) GetUsers(ctx context.Context, activeOnly bool) ([]model.User, error) {
	conn, err := u.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	users, err := u.usersRepo.ListUsers(ctx, conn, activeOnly)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	res := make([]model.User, 0, len(users))

	for _, u := range users {
		res = append(res, model.NewUserFromUserDB(&u))
	}

	return res, nil
}

func (u *Users) LockAccount(ctx context.Context, username string) error {
	//nolint:wrapcheck
	return u.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		udb, err := u.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		udb.Password = model.UserLockedPassword
		udb.UpdatedAt = time.Now()

		if _, err = u.usersRepo.SaveUser(ctx, dbctx, &udb); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})
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
