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

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UsersSrv struct {
	dbi        repository.Database
	usersRepo  repository.Users
	passHasher PasswordHasher
}

func NewUsersSrv(i do.Injector) (*UsersSrv, error) {
	return &UsersSrv{
		dbi:        do.MustInvoke[repository.Database](i),
		usersRepo:  do.MustInvoke[repository.Users](i),
		passHasher: BCryptPasswordHasher{},
	}, nil
}

func (u *UsersSrv) LoginUser(ctx context.Context, username, password string) (*model.User, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	if password == "" {
		return nil, aerr.ErrValidation.WithMsg("password can't be empty")
	}

	user, err := db.InConnectionR(ctx, u.dbi, func(ctx context.Context) (*model.User, error) {
		return u.usersRepo.GetUser(ctx, username)
	})

	common.TraceLazyPrintf(ctx, "LoginUser: user loaded")

	if errors.Is(err, common.ErrNoData) {
		return nil, common.ErrUserNotFound
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if user.Password == model.UserLockedPassword {
		return nil, common.ErrUserAccountLocked
	}

	if !u.passHasher.CheckPassword(password, user.Password) {
		return nil, common.ErrUnauthorized
	}

	common.TraceLazyPrintf(ctx, "LoginUser: password verified")

	return user, nil
}

// CheckUser check is user account valid; return user.
func (u *UsersSrv) CheckUser(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, common.ErrEmptyUsername
	}

	user, err := db.InConnectionR(ctx, u.dbi, func(ctx context.Context) (*model.User, error) {
		return u.usersRepo.GetUser(ctx, username)
	})

	common.TraceLazyPrintf(ctx, "CheckUser: user loaded")

	if errors.Is(err, common.ErrNoData) {
		return nil, common.ErrUserNotFound
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	if user.Password == model.UserLockedPassword {
		return nil, common.ErrUserAccountLocked
	}

	return user, nil
}

func (u *UsersSrv) AddUser(ctx context.Context, cmd *command.NewUserCmd) (command.NewUserCmdResult, error) {
	if cmd == nil {
		panic("cmd is nil")
	}

	res := command.NewUserCmdResult{}

	if err := cmd.Validate(); err != nil {
		return res, aerr.Wrapf(err, "validate user to add failed")
	}

	//nolint:wrapcheck
	return db.InTransactionR(ctx, u.dbi, func(ctx context.Context) (command.NewUserCmdResult, error) {
		// is user exists?
		_, err := u.usersRepo.GetUser(ctx, cmd.UserName)
		switch {
		case errors.Is(err, common.ErrNoData):
			// ok; user not exists
		case err == nil:
			// user exists
			return res, common.ErrUserExists
		default:
			// failed to get user
			return res, aerr.ApplyFor(ErrRepositoryError, err)
		}

		hashedPass, err := u.passHasher.HashPassword(cmd.Password)
		if err != nil {
			return res, aerr.Wrapf(err, "hash password failed")
		}

		udb := model.User{
			UserName: cmd.UserName,
			Password: hashedPass,
			Email:    cmd.Email,
			Name:     cmd.Name,
		}

		uid, err := u.usersRepo.SaveUser(ctx, &udb)
		if err != nil {
			return res, aerr.ApplyFor(ErrRepositoryError, err)
		}

		res.UserID = uid

		return res, nil
	})
}

func (u *UsersSrv) ChangePassword(ctx context.Context, cmd *command.ChangeUserPasswordCmd) error {
	if cmd == nil {
		panic("cmd is nil")
	}

	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate user/password for save failed")
	}

	//nolint: wrapcheck
	return db.InTransaction(ctx, u.dbi, func(ctx context.Context) error {
		// is user exists?
		user, err := u.usersRepo.GetUser(ctx, cmd.UserName)

		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		if cmd.CheckCurrentPass && !u.passHasher.CheckPassword(cmd.CurrentPassword, user.Password) {
			return command.ErrChangePasswordOldNotMatch
		}

		user.Password, err = u.passHasher.HashPassword(cmd.Password)
		if err != nil {
			return aerr.Wrapf(err, "hash password failed")
		}

		if _, err = u.usersRepo.SaveUser(ctx, user); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})
}

func (u *UsersSrv) GetUsers(ctx context.Context, activeOnly bool) ([]model.User, error) {
	users, err := db.InConnectionR(ctx, u.dbi, func(ctx context.Context) ([]model.User, error) {
		return u.usersRepo.ListUsers(ctx, activeOnly)
	})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return users, nil
}

func (u *UsersSrv) LockAccount(ctx context.Context, cmd command.LockAccountCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate account to lock failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, u.dbi, func(ctx context.Context) error {
		udb, err := u.usersRepo.GetUser(ctx, cmd.UserName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		udb.Password = model.UserLockedPassword

		if _, err = u.usersRepo.SaveUser(ctx, udb); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		return nil
	})
}

func (u *UsersSrv) DeleteUser(ctx context.Context, cmd *command.DeleteUserCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate cmd failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, u.dbi, func(ctx context.Context) error {
		user, err := u.usersRepo.GetUser(ctx, cmd.UserName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		if err = u.usersRepo.DeleteUser(ctx, user.ID); err != nil {
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

//-------------------------------------------------------------
