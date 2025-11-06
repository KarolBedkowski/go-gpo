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
	"strings"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
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

func (l *List) Start(ctx context.Context) error {
	if err := l.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	injector := createInjector(ctx)

	db := do.MustInvoke[*db.Database](injector)
	if err := db.Connect(ctx, "sqlite3", l.Database); err != nil {
		return fmt.Errorf("connect to database error: %w", err)
	}

	devSrv := do.MustInvoke[*service.Device](injector)
	subsSrv := do.MustInvoke[*service.Subs](injector)

	switch strings.TrimSpace(l.Object) {
	case "devices":
		return l.listDevices(ctx, devSrv)
	case "subs":
		return l.listSubscriptions(ctx, subsSrv)

	default:
		return aerr.ErrValidation.Clone().WithUserMsg("unknown object for query %q", l.Object)
	}
}

func (l *List) listDevices(ctx context.Context, devsrv *service.Device) error {
	devices, err := devsrv.ListDevices(ctx, l.Username)
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

func (l *List) listSubscriptions(ctx context.Context, subssrv *service.Subs) error {
	subs, err := subssrv.GetUserSubscriptions(ctx, l.Username, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions list error: %w", err)
	}

	for _, s := range subs {
		fmt.Println(s)
	}

	fmt.Printf("\nTotal: %d\n", len(subs))

	return nil
}

func (l *List) validate() error {
	l.Username = strings.TrimSpace(l.Username)
	l.Object = strings.TrimSpace(l.Object)
	l.DeviceID = strings.TrimSpace(l.DeviceID)

	if l.Username == "" {
		return aerr.ErrValidation.Clone().WithUserMsg("username can't be empty")
	}

	return nil
}
