package command

//
// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"slices"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/model"
)

type UpdateDeviceCmd struct {
	UserName   string
	DeviceName string
	DeviceType string
	Caption    string
}

func (u *UpdateDeviceCmd) Validate() error {
	if u.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	if u.DeviceName == "" {
		return aerr.ErrValidation.WithMsg("device name can't be empty")
	}

	if u.DeviceType == "" {
		return aerr.ErrValidation.WithMsg("device type can't be empty")
	}

	if !slices.Contains(model.ValidDevTypes, u.DeviceType) {
		return aerr.ErrValidation.WithMsg("invalid device type %q", u.DeviceType)
	}

	return nil
}

// ------------------------------------------------------

type DeleteDeviceCmd struct {
	UserName   string
	DeviceName string
}

func (u *DeleteDeviceCmd) Validate() error {
	if u.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	if u.DeviceName == "" {
		return aerr.ErrValidation.WithMsg("device name can't be empty")
	}

	return nil
}
