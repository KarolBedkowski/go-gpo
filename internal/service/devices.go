//
// device.go
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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/queries"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type DevicesSrv struct {
	db          *db.Database
	usersRepo   repository.UsersRepository
	devicesRepo repository.DevicesRepository
}

func NewDevicesSrv(i do.Injector) (*DevicesSrv, error) {
	return &DevicesSrv{
		db:          do.MustInvoke[*db.Database](i),
		usersRepo:   do.MustInvoke[repository.UsersRepository](i),
		devicesRepo: do.MustInvoke[repository.DevicesRepository](i),
	}, nil
}

// UpdateDevice update or create device.
func (d *DevicesSrv) UpdateDevice(ctx context.Context, cmd *command.UpdateDeviceCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate dev to update failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, d.db, func(tx repository.DBContext) error {
		user, err := d.usersRepo.GetUser(ctx, tx, cmd.UserName)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		device, err := d.devicesRepo.GetDevice(ctx, tx, user.ID, cmd.DeviceName)
		if errors.Is(err, repository.ErrNoData) {
			// new device
			device = repository.DeviceDB{UserID: user.ID, Name: cmd.DeviceName}
		} else if err != nil {
			return aerr.Wrapf(err, "get device from repo failed")
		}

		device.Caption = cmd.Caption
		device.DevType = cmd.DeviceType

		_, err = d.devicesRepo.SaveDevice(ctx, tx, &device)
		if err != nil {
			return aerr.Wrapf(err, "save device failed")
		}

		return nil
	})
}

// ListDevices return list of user's devices.
func (d *DevicesSrv) ListDevices(ctx context.Context, query *queries.QueryDevices) ([]model.Device, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	devices, err := db.InConnectionR(ctx, d.db, func(conn repository.DBContext) (repository.DevicesDB, error) {
		user, err := d.usersRepo.GetUser(ctx, conn, query.UserName)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		devices, err := d.devicesRepo.ListDevices(ctx, conn, user.ID)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "get devices from db failed")
		}

		return devices, nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	res := make([]model.Device, len(devices))
	for i, d := range devices {
		dev := model.NewDeviceFromDeviceDB(d)
		dev.User = query.UserName
		res[i] = dev
	}

	return res, nil
}

func (d *DevicesSrv) DeleteDevice(ctx context.Context, cmd *command.DeleteDeviceCmd) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, d.db, func(tx repository.DBContext) error {
		user, err := d.usersRepo.GetUser(ctx, tx, cmd.UserName)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		device, err := d.devicesRepo.GetDevice(ctx, tx, user.ID, cmd.DeviceName)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownDevice
		} else if err != nil {
			return aerr.Wrapf(err, "get device from repo failed")
		}

		if err = d.devicesRepo.DeleteDevice(ctx, tx, device.ID); err != nil {
			return aerr.Wrapf(err, "save device failed")
		}

		return nil
	})
}
