//
// list.go
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
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/service"
)

const ListSupportedObjects = "devices, subs"

func newListCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list user objects.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Required: true, Aliases: []string{"u"}},
			&cli.StringFlag{
				Name:     "object",
				Required: true,
				Usage:    "object to list (" + ListSupportedObjects + ")",
				Aliases:  []string{"o"},
			},
			&cli.StringFlag{Name: "device", Aliases: []string{"d"}},
		},
		Action: wrap(listCmd),
	}
}

func listCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	object := clicmd.String("object")
	switch object {
	case "devices":
		return listDevicesCmd(ctx, clicmd, injector)
	case "subs":
		return listSubscriptionsCmd(ctx, clicmd, injector)

	default:
		return aerr.ErrValidation.WithUserMsg("unknown object for query %q", object)
	}
}

func listDevicesCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	devsrv := do.MustInvoke[*service.DevicesSrv](injector)

	devices, err := devsrv.ListDevices(ctx, &query.GetDevicesQuery{UserName: clicmd.String("username")})
	if err != nil {
		return fmt.Errorf("get device list error: %w", err)
	}

	fmt.Printf("%-30s | %-10s | %-30s | %s \n", "Name", "Type", "Caption", "Subscriptions")
	fmt.Println("--------------------------------------------------------------------------------------------")

	for _, d := range devices {
		fmt.Printf("%-30s | %-10s | %-30s | %d \n", d.Name, d.DevType, d.Caption, d.Subscriptions)
	}

	return nil
}

func listSubscriptionsCmd(ctx context.Context, clicmd *cli.Command, injector do.Injector) error {
	subssrv := do.MustInvoke[*service.SubscriptionsSrv](injector)

	subs, err := subssrv.GetUserSubscriptions(
		ctx,
		&query.GetUserSubscriptionsQuery{UserName: clicmd.String("username")},
	)
	if err != nil {
		return fmt.Errorf("get subscriptions list error: %w", err)
	}

	for _, s := range subs {
		fmt.Println(s)
	}

	fmt.Printf("\nTotal: %d\n", len(subs))

	return nil
}
