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

type UsersSrv struct {
	db         *db.Database
	usersRepo  repository.UsersRepository
	passHasher PasswordHasher
}

func NewUsersSrv(i do.Injector) (*UsersSrv, error) {
	db := do.MustInvoke[*db.Database](i)
	repo := do.MustInvoke[repository.UsersRepository](i)

	return &UsersSrv{db, repo, BCryptPasswordHasher{}}, nil
}

func (u *UsersSrv) LoginUser(ctx context.Context, username, password string) (model.User, error) {
	if username == "" {
		return model.User{}, ErrEmptyUsername
	}

	if password == "" {
		return model.User{}, aerr.ErrValidation.WithMsg("password can't be empty")
	}

	//nolint:wrapcheck
	return db.InConnectionR(ctx, u.db, func(conn repository.DBContext) (model.User, error) {
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
	})
}

func (u *UsersSrv) AddUser(ctx context.Context, user *model.NewUser) (int64, error) {
	if user == nil {
		panic("user is nil")
	}

	if err := user.Validate(); err != nil {
		return 0, aerr.Wrapf(err, "validate user to add failed")
	}

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

		uid, err := u.usersRepo.SaveUser(ctx, dbctx, &udb)
		if err != nil {
			return 0, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return uid, nil
	})
}

func (u *UsersSrv) ChangePassword(ctx context.Context, userpass *model.UserPassword) error {
	if userpass == nil {
		panic("userpass is nil")
	}

	if err := userpass.Validate(); err != nil {
		return aerr.Wrapf(err, "validate user/password for save failed")
	}

	//nolint: wrapcheck
	return db.InTransaction(ctx, u.db, func(dbctx repository.DBContext) error {
		// is user exists?
		udb, err := u.usersRepo.GetUser(ctx, dbctx, userpass.Username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		udb.Password, err = u.passHasher.HashPassword(userpass.Password)
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

func (u *UsersSrv) GetUsers(ctx context.Context, activeOnly bool) ([]model.User, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, u.db, func(dbctx repository.DBContext) ([]model.User, error) {
		users, err := u.usersRepo.ListUsers(ctx, dbctx, activeOnly)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		res := make([]model.User, 0, len(users))
		for _, u := range users {
			res = append(res, model.NewUserFromUserDB(&u))
		}

		return res, nil
	})
}

func (u *UsersSrv) LockAccount(ctx context.Context, la model.LockAccount) error {
	if err := la.Validate(); err != nil {
		return aerr.Wrapf(err, "validate account to lock failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, u.db, func(dbctx repository.DBContext) error {
		udb, err := u.usersRepo.GetUser(ctx, dbctx, la.Username)
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

//-------------------------------------------------------------

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
