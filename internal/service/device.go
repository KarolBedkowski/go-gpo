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
	"fmt"
	"slices"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var (
	ErrUnknownUser   = errors.New("unknown user")
	ErrUnknownDevice = errors.New("unknown device")
	ErrInvalidData   = errors.New("invalid data")
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

func (d *Device) UpdateDevice(ctx context.Context, username, deviceid, caption, devtype string) error {
	if username == "" || deviceid == "" || !slices.Contains(model.ValidDevTypes, devtype) {
		return ErrInvalidData
	}

	err := d.db.InTransaction(ctx, func(tx repository.DBContext) error {
		user, err := d.usersRepo.GetUser(ctx, tx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return fmt.Errorf("get user error: %w", err)
		}

		device, err := d.devicesRepo.GetDevice(ctx, tx, user.ID, deviceid)
		if errors.Is(err, repository.ErrNoData) {
			// new device
			device = repository.DeviceDB{UserID: user.ID, Name: deviceid, DevType: "other"}
		} else if err != nil {
			return fmt.Errorf("get device %q for user %q error: %w", deviceid, username, err)
		}

		device.Caption = caption
		device.DevType = devtype

		_, err = d.devicesRepo.SaveDevice(ctx, tx, &device)
		if err != nil {
			return fmt.Errorf("save device error: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update device error: %w", err)
	}

	return nil
}

func (d *Device) ListDevices(ctx context.Context, username string) ([]model.Device, error) {
	conn, err := d.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	user, err := d.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	devices, err := d.devicesRepo.ListDevices(ctx, conn, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	res := make([]model.Device, 0, len(devices))
	for _, d := range devices {
		res = append(res, model.Device{
			User:          username,
			Name:          d.Name,
			DevType:       d.DevType,
			Caption:       d.Caption,
			Subscriptions: d.Subscriptions,
		})
	}

	return res, nil
}
