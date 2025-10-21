// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"time"

	"gitlab.com/kabes/go-gpodder/internal/repository"
)

// FIXME: no json there
type Device struct {
	User          string    `json:"user"`
	Name          string    `json:"id"`
	DevType       string    `json:"type"`
	Caption       string    `json:"caption"`
	Subscriptions int       `json:"subscriptions"`
	UpdatedAt     time.Time `json:"-"`
}

func NewDeviceFromDeviceDB(d *repository.DeviceDB) *Device {
	return &Device{
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
		UpdatedAt:     d.UpdatedAt,
	}
}
