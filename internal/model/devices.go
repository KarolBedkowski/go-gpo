// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"slices"
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var ValidDevTypes = []string{"desktop", "laptop", "mobile", "server", "other"}

//------------------------------------------------------------------------------

type Device struct {
	User          string
	Name          string
	DevType       string
	Caption       string
	Subscriptions int
	UpdatedAt     time.Time
}

func NewDeviceFromDeviceDB(d *repository.DeviceDB) Device {
	return Device{
		Name:          d.Name,
		DevType:       d.DevType,
		Caption:       d.Caption,
		Subscriptions: d.Subscriptions,
		UpdatedAt:     d.UpdatedAt,
	}
}

//------------------------------------------------------------------------------

type UpdatedDevice struct {
	UserName   string
	DeviceName string
	DeviceType string
	Caption    string
}

func NewUpdatedDevice(username, devicename, devicetype, caption string) (UpdatedDevice, error) {
	if devicename == "" {
		return UpdatedDevice{}, aerr.ErrValidation.WithMsg("device name can't be empty")
	}

	if devicetype == "" {
		return UpdatedDevice{}, aerr.ErrValidation.WithMsg("device type can't be empty")
	}

	if !slices.Contains(ValidDevTypes, devicetype) {
		return UpdatedDevice{},
			aerr.ErrValidation.WithMsg("invalid device type %q", devicetype)
	}

	return UpdatedDevice{
		UserName:   username,
		DeviceName: devicename,
		DeviceType: devicetype,
		Caption:    caption,
	}, nil
}
