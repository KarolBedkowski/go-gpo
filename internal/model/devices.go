// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var (
	ValidDevTypes  = []string{"desktop", "laptop", "mobile", "server", "other"}
	ErrInvalidData = errors.New("invalid data")
)

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

func (d Device) Validate() error {
	var errs []string

	if d.Name == "" {
		errs = append(errs, "empty name")
	}

	if d.User == "" {
		errs = append(errs, "empty user")
	}

	if !slices.Contains(ValidDevTypes, d.DevType) {
		errs = append(errs, fmt.Sprintf("invalid device type %q", d.DevType))
	}

	if len(errs) > 0 {
		return aerr.New(strings.Join(errs, ";")).WithTag(aerr.ValidationError)
	}

	return nil
}
