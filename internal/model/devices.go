// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"time"

	"github.com/rs/zerolog"
)

//------------------------------------------------------------------------------

type Device struct {
	ID            int64
	Name          string
	DevType       string
	Caption       string
	Subscriptions int
	UpdatedAt     time.Time
	LastSeenAt    time.Time

	User *User
}

func (d *Device) MarshalZerologObject(event *zerolog.Event) {
	event.Int64("id", d.ID).
		Str("name", d.Name).
		Str("type", d.DevType).
		Str("caption", d.Caption).
		Time("updated_at", d.UpdatedAt).
		Time("last_seen_at", d.LastSeenAt).
		Int("subscriptions", d.Subscriptions)

	if d.User != nil {
		event.Object("user", d.User)
	}
}

//------------------------------------------------------------------------------

type Devices []Device

func (d Devices) ToMap() map[string]Device {
	devices := make(map[string]Device)

	for _, dev := range d {
		devices[dev.Name] = dev
	}

	return devices
}

func (d Devices) ToIDsMap() map[string]int64 {
	devices := make(map[string]int64)

	for _, dev := range d {
		devices[dev.Name] = dev.ID
	}

	return devices
}
