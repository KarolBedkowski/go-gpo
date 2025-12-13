package command

//
// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type UpdateDeviceCmd struct {
	UserName   string
	DeviceName string
	DeviceType string
	Caption    string
}

func (u *UpdateDeviceCmd) Validate() error {
	if u.UserName == "" {
		return common.ErrInvalidUser.WithUserMsg("user name can't be empty")
	}

	if !validators.IsValidUserName(u.UserName) {
		return common.ErrInvalidUser
	}

	if u.DeviceName == "" {
		return common.ErrInvalidDevice.WithUserMsg("device name can't be empty")
	}

	if !validators.IsValidDevName(u.DeviceName) {
		return common.ErrInvalidDevice
	}

	if u.DeviceType == "" {
		return aerr.ErrValidation.WithUserMsg("device type can't be empty")
	}

	if !validators.IsValidDevType(u.DeviceType) {
		return aerr.ErrValidation.WithUserMsg("invalid device type %q", u.DeviceType)
	}

	return nil
}

// ------------------------------------------------------

type DeleteDeviceCmd struct {
	UserName   string
	DeviceName string
}

func (u *DeleteDeviceCmd) Validate() error {
	if !validators.IsValidUserName(u.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if !validators.IsValidDevName(u.DeviceName) {
		return common.ErrInvalidDevice.WithUserMsg("invalid device name")
	}

	return nil
}
