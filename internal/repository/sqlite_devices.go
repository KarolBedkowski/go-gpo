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
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

func (s SqliteRepository) GetDevice(
	ctx context.Context,
	dbctx DBContext,
	userid int64,
	devicename string,
) (DeviceDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_dev").Logger()
	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("get device")

	device := DeviceDB{}
	err := dbctx.GetContext(ctx, &device,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=? and name=?",
		userid, devicename)

	if errors.Is(err, sql.ErrNoRows) {
		return device, ErrNoData
	} else if err != nil {
		return device, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid, "device_name", devicename)
	}

	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("count subscriptions")

	err = dbctx.GetContext(ctx, &device.Subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid,
	)
	if err != nil {
		return device, aerr.Wrapf(err, "count subscriptions failed").WithMeta("user_id", userid)
	}

	return device, nil
}

func (s SqliteRepository) SaveDevice(ctx context.Context, dbctx DBContext, device *DeviceDB) (int64, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_dev").Logger()

	if device.ID == 0 {
		logger.Debug().Object("device", device).Msg("insert device")

		res, err := dbctx.ExecContext(ctx,
			"INSERT INTO devices (user_id, name, dev_type, caption, updated_at, created_at) VALUES(?, ?, ?, ?, ?, ?)",
			device.UserID, device.Name, device.DevType, device.Caption, time.Now(), time.Now())
		if err != nil {
			return 0, aerr.Wrapf(err, "insert device failed")
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, aerr.Wrapf(err, "get last id device failed").
				WithMeta("device_name", device.Name, "user_id", device.UserID)
		}

		return id, nil
	}

	// update
	logger.Debug().Object("device", device).Msg("update device")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE devices SET dev_type=?, caption=?, updated_at=? WHERE id=?",
		device.DevType, device.Caption, time.Now(), device.ID)
	if err != nil {
		return device.ID, aerr.Wrapf(err, "update device failed").WithMeta("device_id", device.ID)
	}

	return device.ID, nil
}

func (s SqliteRepository) ListDevices(ctx context.Context, dbctx DBContext, userid int64) (DevicesDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_dev").Logger()
	logger.Debug().Int64("user_id", userid).Msg("list devices - count subscriptions")

	// all device have the same number of subscriptions
	var subscriptions int

	err := dbctx.GetContext(ctx, &subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "count subscriptions error").WithMeta("user_id", userid)
	}

	logger.Debug().Int64("user_id", userid).Msg("list devices")

	res := []*DeviceDB{}

	err = dbctx.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, ? as subscriptions, created_at, updated_at "+
			"FROM devices WHERE user_id=? "+
			"ORDER BY name",
		subscriptions, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid)
	}

	return res, nil
}

func (s SqliteRepository) DeleteDevice(ctx context.Context, dbctx DBContext, deviceid int64) error {
	logger := log.Ctx(ctx).With().Logger()
	logger.Debug().Int64("device_id", deviceid).Msg("delete device")

	_, err := dbctx.ExecContext(ctx, "UPDATE episodes SET device_id=NULL WHERE device_id=?", deviceid)
	if err != nil {
		return aerr.Wrapf(err, "delete device failed")
	}

	_, err = dbctx.ExecContext(ctx, "DELETE FROM devices where id=?", deviceid)
	if err != nil {
		return aerr.Wrapf(err, "delete device failed")
	}

	return nil
}
