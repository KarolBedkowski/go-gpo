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

func (s sqliteRepository) GetDevice(ctx context.Context, userid int64, devicename string) (DeviceDB, error) {
	device := DeviceDB{}
	err := s.db.GetContext(ctx, &device,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=? and name=?",
		userid, devicename)

	if errors.Is(err, sql.ErrNoRows) {
		return device, ErrNoData
	} else if err != nil {
		return device, fmt.Errorf("query device error: %w", err)
	}

	err = s.db.GetContext(ctx, &device.Subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid,
	)
	if err != nil {
		return device, fmt.Errorf("count subscriptions error: %w", err)
	}

	return device, nil
}

func (s sqliteRepository) SaveDevice(ctx context.Context, device *DeviceDB) (int64, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Object("device", device).Msg("update device")

	if device.ID == 0 {
		res, err := s.db.ExecContext(ctx,
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
	_, err := s.db.ExecContext(ctx,
		"UPDATE devices SET dev_type=?, caption=?, updated_at=current_timestamp WHERE id=?",
		device.DevType, device.Caption, device.ID)
	if err != nil {
		return device.ID, fmt.Errorf("update device error: %w", err)
	}

	return device.ID, nil
}

func (s sqliteRepository) ListDevices(ctx context.Context, userid int64) (DevicesDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	// all device have the same number of subscriptions
	var subscriptions int

	err := s.db.GetContext(ctx, &subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions error: %w", err)
	}

	res := []*DeviceDB{}

	err = s.db.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, ? as subscriptions, created_at, updated_at "+
			"FROM devices WHERE user_id=?",
		subscriptions, userid)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}
