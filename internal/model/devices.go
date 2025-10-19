// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"time"

	"gitlab.com/kabes/go-gpodder/internal/repository"
)

type Device struct {
	User          string
	Name          string `json:"id"`
	DevType       string
	Caption       string
	Subscriptions int
	UpdatedAt     time.Time
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
