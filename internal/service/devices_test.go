package service

//
// users_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func TestDevice(t *testing.T) {
	ctx := context.Background()
	i := prepareTests(ctx, t)
	deviceSrv := do.MustInvoke[*DevicesSrv](i)
	_ = prepareTestUser(ctx, t, i, "test")

	// add device
	udev := model.NewUpdatedDevice("test", "dev1", "mobile", "device caption")

	err := deviceSrv.UpdateDevice(ctx, &udev)
	assert.NoErr(t, err)

	devices, err := deviceSrv.ListDevices(ctx, "test")
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 1)

	// add another
	udev2 := model.NewUpdatedDevice("test", "dev2", "desktop", "device 2 caption")

	err = deviceSrv.UpdateDevice(ctx, &udev2)
	assert.NoErr(t, err)

	devices, err = deviceSrv.ListDevices(ctx, "test")
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)
	assert.Equal(t, devices[0].Name, udev.DeviceName)
	assert.Equal(t, devices[0].User, "test")
	assert.Equal(t, devices[0].Caption, udev.Caption)
	assert.Equal(t, devices[0].DevType, udev.DeviceType)
	assert.Equal(t, devices[1].Name, udev2.DeviceName)
	assert.Equal(t, devices[1].User, "test")
	assert.Equal(t, devices[1].Caption, udev2.Caption)
	assert.Equal(t, devices[1].DevType, udev2.DeviceType)

	// update
	udev3 := model.NewUpdatedDevice("test", "dev1", "other", "device 1 new caption")

	err = deviceSrv.UpdateDevice(ctx, &udev3)
	assert.NoErr(t, err)

	devices, err = deviceSrv.ListDevices(ctx, "test")
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)
	assert.Equal(t, devices[0].Name, udev3.DeviceName)
	assert.Equal(t, devices[0].User, "test")
	assert.Equal(t, devices[0].Caption, udev3.Caption)
	assert.Equal(t, devices[0].DevType, udev3.DeviceType)
	assert.Equal(t, devices[1].Name, udev2.DeviceName)
	assert.Equal(t, devices[1].User, "test")
	assert.Equal(t, devices[1].Caption, udev2.Caption)
	assert.Equal(t, devices[1].DevType, udev2.DeviceType)
}
