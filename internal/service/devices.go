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
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type DevicesSrv struct {
	dbi         repository.Database
	usersRepo   repository.Users
	devicesRepo repository.Devices
}

func NewDevicesSrv(i do.Injector) (*DevicesSrv, error) {
	return &DevicesSrv{
		dbi:         do.MustInvoke[repository.Database](i),
		usersRepo:   do.MustInvoke[repository.Users](i),
		devicesRepo: do.MustInvoke[repository.Devices](i),
	}, nil
}

// UpdateDevice update or create device.
func (d *DevicesSrv) UpdateDevice(ctx context.Context, cmd *command.UpdateDeviceCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate dev to update failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, d.dbi, func(ctx context.Context) error {
		user, err := d.usersRepo.GetUser(ctx, cmd.UserName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "user loaded")

		device, err := d.devicesRepo.GetDevice(ctx, user.ID, cmd.DeviceName)
		if errors.Is(err, common.ErrNoData) {
			// new device
			device = &model.Device{Name: cmd.DeviceName, User: &model.User{ID: user.ID}}
		} else if err != nil {
			return aerr.Wrapf(err, "get device from repo failed")
		}

		common.TraceLazyPrintf(ctx, "device loaded")

		device.Caption = cmd.Caption
		device.DevType = cmd.DeviceType
		device.User = user

		_, err = d.devicesRepo.SaveDevice(ctx, device)
		if err != nil {
			return aerr.Wrapf(err, "save device failed")
		}

		common.TraceLazyPrintf(ctx, "device saveed")

		return nil
	})
}

// ListDevices return list of user's devices.
func (d *DevicesSrv) ListDevices(ctx context.Context, query *query.GetDevicesQuery) (model.Devices, error) {
	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validate query failed")
	}

	devices, err := db.InConnectionR(ctx, d.dbi, func(ctx context.Context) ([]model.Device, error) {
		user, err := d.usersRepo.GetUser(ctx, query.UserName)
		if errors.Is(err, common.ErrNoData) {
			return nil, common.ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "user loaded")

		devices, err := d.devicesRepo.ListDevices(ctx, user.ID)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "get devices from db failed")
		}

		common.TraceLazyPrintf(ctx, "devices loaded")

		return devices, nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return devices, nil
}

func (d *DevicesSrv) DeleteDevice(ctx context.Context, cmd *command.DeleteDeviceCmd) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate cmd failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, d.dbi, func(ctx context.Context) error {
		user, err := d.usersRepo.GetUser(ctx, cmd.UserName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownUser
		} else if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		common.TraceLazyPrintf(ctx, "user loaded")

		device, err := d.devicesRepo.GetDevice(ctx, user.ID, cmd.DeviceName)
		if errors.Is(err, common.ErrNoData) {
			return common.ErrUnknownDevice
		} else if err != nil {
			return aerr.Wrapf(err, "get device from repo failed")
		}

		common.TraceLazyPrintf(ctx, "device loaded")

		if err = d.devicesRepo.DeleteDevice(ctx, device.ID); err != nil {
			return aerr.Wrapf(err, "save device failed")
		}

		common.TraceLazyPrintf(ctx, "device deleted")

		return nil
	})
}
