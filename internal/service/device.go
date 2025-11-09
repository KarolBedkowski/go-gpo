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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Device struct {
	db          *db.Database
	usersRepo   repository.UsersRepository
	devicesRepo repository.DevicesRepository
}

func NewDeviceServiceI(i do.Injector) (*Device, error) {
	return &Device{
		db:          do.MustInvoke[*db.Database](i),
		usersRepo:   do.MustInvoke[repository.UsersRepository](i),
		devicesRepo: do.MustInvoke[repository.DevicesRepository](i),
	}, nil
}

// UpdateDevice update or create device.
func (d *Device) UpdateDevice(ctx context.Context, updateddev *model.UpdatedDevice) error {
	//nolint:wrapcheck
	return d.db.InTransaction(ctx, func(tx repository.DBContext) error {
		return d.updateDevice(ctx, tx, updateddev)
	})
}

// ListDevices return list of user's devices.
func (d *Device) ListDevices(ctx context.Context, username string) ([]model.Device, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, d.db, func(conn repository.DBContext) ([]model.Device, error) {
		return d.listDevices(ctx, conn, username)
	})
}

func (d *Device) updateDevice(ctx context.Context, tx repository.DBContext, updateddev *model.UpdatedDevice) error {
	user, err := d.usersRepo.GetUser(ctx, tx, updateddev.UserName)
	if errors.Is(err, repository.ErrNoData) {
		return ErrUnknownUser
	} else if err != nil {
		return aerr.ApplyFor(ErrRepositoryError, err)
	}

	device, err := d.devicesRepo.GetDevice(ctx, tx, user.ID, updateddev.DeviceName)
	if errors.Is(err, repository.ErrNoData) {
		// new device
		device = repository.DeviceDB{UserID: user.ID, Name: updateddev.DeviceName, DevType: "other"}
	} else if err != nil {
		return aerr.Wrapf(err, "get device from repo failed")
	}

	device.Caption = updateddev.Caption
	device.DevType = updateddev.DeviceType

	_, err = d.devicesRepo.SaveDevice(ctx, tx, &device)
	if err != nil {
		return aerr.Wrapf(err, "save device failed")
	}

	return nil
}

func (d *Device) listDevices(ctx context.Context, conn repository.DBContext, username string) ([]model.Device, error) {
	user, err := d.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	devices, err := d.devicesRepo.ListDevices(ctx, conn, user.ID)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err, "get devices from db failed")
	}

	res := make([]model.Device, len(devices))
	for i, d := range devices {
		res[i] = model.NewDeviceFromDeviceDB(d)
	}

	return res, nil
}
