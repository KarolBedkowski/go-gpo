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
	"time"

	"gitlab.com/kabes/go-gpodder/internal/repository"
	"gitlab.com/kabes/go-gpodder/internal/service"
)

type List struct {
	Database string
	Username string
	DeviceID string
	Object   string
}

func (a *List) Start(ctx context.Context) error {
	re := &repository.Database{}
	if err := re.Connect(ctx, "sqlite3", a.Database+"?_fk=true"); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	switch a.Object {
	case "devices":
		return a.listDevices(ctx, re)
	case "subs":
		return a.listSubscriptions(ctx, re)

	default:
		return fmt.Errorf("unknown object %q", a.Object) //nolint:err113
	}
}

func (a *List) listDevices(ctx context.Context, re *repository.Database) error {
	devsrv := service.NewDeviceService(re)

	devices, err := devsrv.ListDevices(ctx, a.Username)
	if err != nil {
		return fmt.Errorf("get device list error: %w", err)
	}

	for _, d := range devices {
		fmt.Printf("%-30s | %-10s | %-30s | %d \n", d.Name, d.DevType, d.Caption, d.Subscriptions)
	}

	return nil
}

func (a *List) listSubscriptions(ctx context.Context, re *repository.Database) error {
	subssrv := service.NewSubssService(re)

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
