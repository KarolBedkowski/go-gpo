//
// list.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/service"
)

type List struct {
	Database string
	Username string
	DeviceID string
	Object   string
}

const ListSupportedObjects = "devices, subs"

func (a *List) Start(ctx context.Context) error {
	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", a.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	devSrv := do.MustInvoke[*service.Device](injector)
	subsSrv := do.MustInvoke[*service.Subs](injector)

	switch a.Object {
	case "devices":
		return a.listDevices(ctx, devSrv)
	case "subs":
		return a.listSubscriptions(ctx, subsSrv)

	default:
		return fmt.Errorf("unknown object for query %q", a.Object) //nolint:err113
	}
}

func (a *List) listDevices(ctx context.Context, devsrv *service.Device) error {
	devices, err := devsrv.ListDevices(ctx, a.Username)
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

func (a *List) listSubscriptions(ctx context.Context, subssrv *service.Subs) error {
	subs, err := subssrv.GetUserSubscriptions(ctx, a.Username, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions list error: %w", err)
	}

	for _, s := range subs {
		fmt.Println(s)
	}

	fmt.Printf("\nTotal: %d\n", len(subs))

	return nil
}
