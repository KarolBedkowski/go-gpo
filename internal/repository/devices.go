package repository

//
// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

func (t *Transaction) GetDevice(ctx context.Context, userid int64, devicename string) (DeviceDB, error) {
	device := DeviceDB{}
	err := t.tx.QueryRowxContext(ctx,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=? and name=?", userid, devicename).
		StructScan(&device)

	if errors.Is(err, sql.ErrNoRows) {
		return device, ErrNoData
	} else if err != nil {
		return device, fmt.Errorf("query device error: %w", err)
	}

	err = t.tx.GetContext(
		ctx,
		&device.Subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid,
	)
	if err != nil {
		return device, fmt.Errorf("count subscriptions error: %w", err)
	}

	return device, nil
}

func (t *Transaction) getUserDevices(ctx context.Context, userid int64) (DevicesDB, error) {
	res := []*DeviceDB{}

	err := t.tx.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userid)
	if err != nil {
		return nil, fmt.Errorf("query device error: %w", err)
	}

	// all device have the same number of subscriptions
	var subscriptions int

	err = t.tx.GetContext(ctx, &subscriptions, "SELECT count(*) FROM podcasts where user_id=? and subscribed", userid)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions error: %w", err)
	}

	for _, t := range res {
		t.Subscriptions = subscriptions
	}

	return res, nil
}

func (t *Transaction) SaveDevice(ctx context.Context, device *DeviceDB) (int64, error) {
	return t.saveDevice(ctx, device)
}

func (t *Transaction) saveDevice(ctx context.Context, device *DeviceDB) (int64, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("device", device).Msg("update device")

	if device.ID == 0 {
		res, err := t.tx.ExecContext(ctx,
			"INSERT INTO devices (user_id, name, dev_type, caption) VALUES(?, ?, ?, ?)",
			device.UserID, device.Name, device.DevType, device.Caption)
		if err != nil {
			return 0, fmt.Errorf("insert new device error: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id error: %w", err)
		}

		return id, nil
	}

	// update
	_, err := t.tx.ExecContext(ctx,
		"UPDATE devices SET dev_type=?, caption=?, updated_at=current_timestamp WHERE id=?",
		device.DevType, device.Caption, device.ID)
	if err != nil {
		return device.ID, fmt.Errorf("update device error: %w", err)
	}

	return device.ID, nil
}

func (t *Transaction) ListDevices(ctx context.Context, userid int64) (DevicesDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	res := []*DeviceDB{}

	err := t.tx.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userid)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}

func (t *Transaction) createNewDevice(ctx context.Context, userid int64, devicename string) (int64, error) {
	dev := DeviceDB{UserID: userid, Name: devicename, DevType: "computer"}

	id, err := t.saveDevice(ctx, &dev)
	if err != nil {
		return 0, err
	}

	return id, nil
}
