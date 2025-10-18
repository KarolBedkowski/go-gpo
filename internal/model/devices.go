// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "time"

type Device struct {
	User          string
	Name          string `json:"id"`
	DevType       string
	Caption       string
	Subscriptions int
	UpdatedAt     time.Time
}

func NewDeviceFromDeviceDB(d *DeviceDB) *Device {
	return &Device{
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
		UpdatedAt:     d.UpdatedAt,
	}
}

type DeviceDB struct {
	ID            int       `db:"id"`
	UserID        int       `db:"user_id"`
	Name          string    `db:"name"`
	DevType       string    `db:"dev_type"`
	Caption       string    `db:"caption"`
	Subscriptions int       `db:"subscriptions"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
