//
// adduser.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//---------------------------------------------------------------------

type UpdateDevice struct {
	Database      string
	Username      string
	DeviceName    string
	DeviceType    string
	DeviceCaption string
}

func (u *UpdateDevice) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", u.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	devsrv := do.MustInvoke[*service.DevicesSrv](injector)

	updateddev := model.UpdatedDevice{
		UserName:   u.Username,
		DeviceName: u.DeviceName,
		DeviceType: u.DeviceType,
		Caption:    u.DeviceCaption,
	}
	if err := devsrv.UpdateDevice(ctx, &updateddev); err != nil {
		return fmt.Errorf("update device error: %w", err)
	}

	fmt.Printf("Device updated")

	return nil
}

//---------------------------------------------------------------------

type DeleteDevice struct {
	Database   string
	Username   string
	DeviceName string
}

func (d *DeleteDevice) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", d.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	devsrv := do.MustInvoke[*service.DevicesSrv](injector)

	if err := devsrv.DeleteDevice(ctx, d.Username, d.DeviceName); err != nil {
		return fmt.Errorf("delete device error: %w", err)
	}

	fmt.Printf("Device updated")

	return nil
}
