package service

//
// users_test.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"

	"gitlab.com/kabes/go-gpo/internal/assert"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/query"
)

func TestDevice(t *testing.T) {
	ctx, i := prepareTests(t)
	deviceSrv := do.MustInvoke[*DevicesSrv](i)
	_ = prepareTestUser(ctx, t, i, "test")

	// add device
	cmd1 := command.UpdateDeviceCmd{
		UserName:   "test",
		DeviceName: "dev1",
		DeviceType: "mobile",
		Caption:    "device caption",
	}
	err := deviceSrv.UpdateDevice(ctx, &cmd1)
	assert.NoErr(t, err)

	devices, err := deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: "test"})
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 1)

	// add another
	cmd2 := command.UpdateDeviceCmd{
		UserName:   "test",
		DeviceName: "dev2",
		DeviceType: "desktop",
		Caption:    "device 2 caption",
	}
	err = deviceSrv.UpdateDevice(ctx, &cmd2)
	assert.NoErr(t, err)

	devices, err = deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: "test"})
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)
	assert.Equal(t, devices[0].Name, cmd1.DeviceName)
	assert.Equal(t, devices[0].User, "test")
	assert.Equal(t, devices[0].Caption, cmd1.Caption)
	assert.Equal(t, devices[0].DevType, cmd1.DeviceType)
	assert.Equal(t, devices[1].Name, cmd2.DeviceName)
	assert.Equal(t, devices[1].User, "test")
	assert.Equal(t, devices[1].Caption, cmd2.Caption)
	assert.Equal(t, devices[1].DevType, cmd2.DeviceType)

	// update
	cmd3 := command.UpdateDeviceCmd{
		UserName:   "test",
		DeviceName: "dev1",
		DeviceType: "other",
		Caption:    "device 1 new caption",
	}

	err = deviceSrv.UpdateDevice(ctx, &cmd3)
	assert.NoErr(t, err)

	devices, err = deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: "test"})
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 2)
	assert.Equal(t, devices[0].Name, cmd3.DeviceName)
	assert.Equal(t, devices[0].User, "test")
	assert.Equal(t, devices[0].Caption, cmd3.Caption)
	assert.Equal(t, devices[0].DevType, cmd3.DeviceType)
	assert.Equal(t, devices[1].Name, cmd2.DeviceName)
	assert.Equal(t, devices[1].User, "test")
	assert.Equal(t, devices[1].Caption, cmd2.Caption)
	assert.Equal(t, devices[1].DevType, cmd2.DeviceType)
}

func TestDeleteDevice(t *testing.T) {
	ctx, i := prepareTests(t)
	_ = prepareTestUser(ctx, t, i, "user1")
	_ = prepareTestUser(ctx, t, i, "user2")
	prepareTestDevice(ctx, t, i, "user1", "dev1")
	prepareTestDevice(ctx, t, i, "user1", "dev2")
	prepareTestSub(
		ctx, t, i, "user1", "dev1", "http://example.com/p1", "http://example.com/p2",
	)

	deviceSrv := do.MustInvoke[*DevicesSrv](i)
	err := deviceSrv.DeleteDevice(ctx, &command.DeleteDeviceCmd{UserName: "user1", DeviceName: "dev1"})
	assert.NoErr(t, err)

	devices, err := deviceSrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: "user1"})
	assert.NoErr(t, err)
	assert.Equal(t, len(devices), 1)
	assert.Equal(t, devices[0].Name, "dev2")
}
