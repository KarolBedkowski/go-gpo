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

	apperrors "gitlab.com/kabes/go-gpodder/internal/errors"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

var (
	ErrUnknownUser   = errors.New("unknown user")
	ErrUnknownDevice = errors.New("unknown device")
)

type Device struct {
	repo *repository.Database
}

func NewDeviceService(repo *repository.Database) *Device {
	return &Device{repo}
}

func (d *Device) UpdateDevice(ctx context.Context, username, deviceid, caption, devtype string) error {
	if username == "" || deviceid == "" || !slices.Contains(model.ValidDevTypes, devtype) {
		return apperrors.NewAppError("invalid data").WithCategory(apperrors.ValidationError)
	}

	err := d.repo.InTransaction(ctx, func(tx repository.DBContext) error {
		repo := d.repo.GetRepository(tx)

		user, err := repo.GetUser(ctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return ErrUnknownUser
		} else if err != nil {
			return fmt.Errorf("get user error: %w", err)
		}

		device, err := repo.GetDevice(ctx, user.ID, deviceid)
		if errors.Is(err, repository.ErrNoData) {
			// new device
			device = repository.DeviceDB{UserID: user.ID, Name: deviceid, DevType: "other"}
		} else if err != nil {
			return fmt.Errorf("get device %q for user %q error: %w", deviceid, username, err)
		}

		device.Caption = caption
		device.DevType = devtype

		_, err = repo.SaveDevice(ctx, &device)
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
	conn, err := d.repo.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := d.repo.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	devices, err := repo.ListDevices(ctx, user.ID)
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
