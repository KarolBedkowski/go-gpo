package sqlite

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
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
)

func (s SqliteRepository) GetDevice(
	ctx context.Context,
	userid int64,
	devicename string,
) (*model.Device, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("get device")

	dbctx := db.MustCtx(ctx)

	device := DeviceDB{}
	err := dbctx.GetContext(ctx, &device,
		`
		SELECT d.id, d.user_id, d.name, d.dev_type, d.caption, d.created_at, d.updated_at,
				u.name as user_name, u.username as user_username
		FROM devices d
		JOIN users u ON u.ID = d.user_id
		WHERE d.user_id=? and d.name=?
		`,
		userid, devicename)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, common.ErrNoData
	} else if err != nil {
		return nil, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid, "device_name", devicename)
	}

	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("count subscriptions")

	err = dbctx.GetContext(ctx, &device.Subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid,
	)
	if err != nil {
		return nil, aerr.Wrapf(err, "count subscriptions failed").WithMeta("user_id", userid)
	}

	logger.Debug().Int("subs", device.Subscriptions).Msg("count subscriptions finished")

	return device.toModel(), nil
}

func (s SqliteRepository) SaveDevice(ctx context.Context, device *model.Device) (int64, error) {
	logger := log.Ctx(ctx)
	dbctx := db.MustCtx(ctx)

	logger.Debug().Object("device", device).Msg("save device")

	if device.ID == 0 {
		logger.Debug().Object("device", device).Msg("insert device")

		now := time.Now().UTC()

		res, err := dbctx.ExecContext(ctx,
			"INSERT INTO devices (user_id, name, dev_type, caption, updated_at, created_at, last_seen_at) "+
				"VALUES(?, ?, ?, ?, ?, ?, ?)",
			device.User.ID, device.Name, device.DevType, device.Caption, now, now, now)
		if err != nil {
			return 0, aerr.Wrapf(err, "insert device failed")
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, aerr.Wrapf(err, "get last id device failed").
				WithMeta("device_name", device.Name, "user_id", device.User.ID)
		}

		return id, nil
	}

	// update
	logger.Debug().Object("device", device).Msg("update device")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE devices SET dev_type=?, caption=?, updated_at=? WHERE id=?",
		device.DevType, device.Caption, time.Now().UTC(), device.ID)
	if err != nil {
		return device.ID, aerr.Wrapf(err, "update device failed").WithMeta("device_id", device.ID)
	}

	return device.ID, nil
}

func (s SqliteRepository) ListDevices(ctx context.Context, userid int64) ([]model.Device, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("list devices - count subscriptions")

	// all device have the same number of subscriptions
	var subscriptions int

	dbctx := db.MustCtx(ctx)

	err := dbctx.GetContext(ctx, &subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "count subscriptions error").WithMeta("user_id", userid)
	}

	logger.Debug().Int64("user_id", userid).Msg("list devices")

	devices := []DeviceDB{}

	err = dbctx.SelectContext(ctx, &devices, `
			SELECT d.id, d.user_id, d.name, d.dev_type, d.caption, ? as subscriptions,
				d.created_at, d.updated_at, d.last_seen_at,
				u.name as user_name, u.username as user_username
			FROM devices d
			JOIN users u ON u.id = d.user_id
			WHERE user_id=?
			ORDER BY d.name`,
		subscriptions, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid)
	}

	return devicesFromDb(devices), nil
}

func (s SqliteRepository) DeleteDevice(ctx context.Context, deviceid int64) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("device_id", deviceid).Msg("delete device")

	dbctx := db.MustCtx(ctx)

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

func (s SqliteRepository) MarkSeen(ctx context.Context, ts time.Time, deviceid ...int64) error {
	logger := log.Ctx(ctx)
	logger.Debug().Ints64("device_id", deviceid).Msgf("mark device seen at: %s", ts)

	dbctx := db.MustCtx(ctx)

	for _, did := range deviceid {
		_, err := dbctx.ExecContext(ctx, "UPDATE devices SET last_seen_at=? WHERE id=?", ts, did)
		if err != nil {
			return aerr.Wrapf(err, "update device failed")
		}
	}

	return nil
}
