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

func (d *Device) UpdateDevice(ctx context.Context, username, deviceID, caption, devtype string) error {
	user, err := d.repo.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return ErrUnknownUser
	}

	device, err := d.repo.GetDevice(ctx, user.ID, deviceID)
	if err != nil {
		return fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		// new device
		device = &model.DeviceDB{
			UserID:  user.ID,
			Name:    deviceID,
			Caption: caption,
			DevType: devtype,
		}
	} else {
		device.Caption = caption
		device.DevType = devtype
	}

	_, err = d.repo.SaveDevice(ctx, device)
	if err != nil {
		return fmt.Errorf("save device error: %w", err)
	}

	return nil
}

func (d *Device) ListDevices(ctx context.Context, username string) ([]*model.Device, error) {
	user, err := d.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	devices, err := d.repo.ListDevices(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	res := make([]*model.Device, 0, len(devices))
	for _, d := range devices {
		res = append(res, &model.Device{
			User:          username,
			Name:          d.Name,
			DevType:       d.DevType,
			Caption:       d.Caption,
			Subscriptions: d.Subscriptions,
		})
	}

	return res, nil
}
