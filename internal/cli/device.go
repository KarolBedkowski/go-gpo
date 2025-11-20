//
// adduser.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cli

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/service"
)

//---------------------------------------------------------------------

func NewUpdateDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "add or update device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "device", Required: true, Aliases: []string{"d"}},
			&cli.StringFlag{
				Name: "type", Required: false, Aliases: []string{"t"}, Value: "mobile",
				Usage: "device type (desktop, laptop, mobile, server, other)",
			},
			&cli.StringFlag{Name: "caption", Required: false, Aliases: []string{"c"}},
		},
		Action: wrap(updateDeviceCmd),
	}
}

func updateDeviceCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	devsrv := do.MustInvoke[*service.DevicesSrv](injector)

	cmd := command.UpdateDeviceCmd{
		UserName:   clicmd.String("username"),
		DeviceName: clicmd.String("device"),
		DeviceType: clicmd.String("type"),
		Caption:    clicmd.String("caption"),
	}
	if err := devsrv.UpdateDevice(ctx, &cmd); err != nil {
		return fmt.Errorf("update device error: %w", err)
	}

	fmt.Printf("Device updated")

	return nil
}

//---------------------------------------------------------------------

func NewDeleteDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{Name: "device", Required: true, Aliases: []string{"d"}},
		},
		Action: wrap(deleteDeviceCmd),
	}
}

func deleteDeviceCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	devsrv := do.MustInvoke[*service.DevicesSrv](injector)

	cmd := command.DeleteDeviceCmd{UserName: clicmd.String("username"), DeviceName: clicmd.String("device")}
	if err := devsrv.DeleteDevice(ctx, &cmd); err != nil {
		return fmt.Errorf("delete device error: %w", err)
	}

	fmt.Printf("Device updated")

	return nil
}

//-----------

func NewListDeviceCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
		},
		Action: wrap(listDevicesCmd),
	}
}
