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

	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

var (
	ErrUnknownUser   = errors.New("unknown user")
	ErrUnknownDevice = errors.New("unknown device")
)

type Device struct {
	repo *repository.Repository
}

func NewDeviceService(repo *repository.Repository) *Device {
	return &Device{repo}
}

func (d *Device) UpdateDevice(ctx context.Context, username, deviceid, caption, devtype string) error {
	tx, err := d.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return ErrUnknownUser
	} else if err != nil {
		return fmt.Errorf("get user error: %w", err)
	}

	device, err := tx.GetDevice(ctx, user.ID, deviceid)
	if errors.Is(err, repository.ErrNoData) {
		// new device
		device = repository.DeviceDB{UserID: user.ID, Name: deviceid}
	} else if err != nil {
		return fmt.Errorf("get device %q for user %q error: %w", deviceid, username, err)
	}

	device.Caption = caption
	device.DevType = devtype

	_, err = tx.SaveDevice(ctx, &device)
	if err != nil {
		return fmt.Errorf("save device error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

func (d *Device) ListDevices(ctx context.Context, username string) ([]model.Device, error) {
	tx, err := d.repo.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	devices, err := tx.ListDevices(ctx, user.ID)
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
