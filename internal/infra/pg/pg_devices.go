package pg

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

func (s Repository) GetDevice(
	ctx context.Context,
	userid int64,
	devicename string,
) (*model.Device, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("get device")

	dbctx := db.MustCtx(ctx)

	device := DeviceDB{}
	err := dbctx.GetContext(ctx, &device, `
		SELECT d.id, d.user_id, d.name, d.dev_type, d.caption, d.created_at, d.updated_at,
				u.id AS "user.id", u.name AS "user.name", u.username AS "user.username"
		FROM devices d
		JOIN users u ON u.ID = d.user_id
		WHERE d.user_id=$1 AND d.name=$2`,
		userid, devicename)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, common.ErrNoData
	} else if err != nil {
		return nil, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid, "device_name", devicename)
	}

	logger.Debug().Int64("user_id", userid).Str("device_name", devicename).Msg("count subscriptions")

	err = dbctx.GetContext(ctx, &device.Subscriptions,
		"SELECT count(*) FROM podcasts WHERE user_id=$1 AND subscribed",
		userid,
	)
	if err != nil {
		return nil, aerr.Wrapf(err, "count subscriptions failed").WithMeta("user_id", userid)
	}

	logger.Debug().Int("subs", device.Subscriptions).Object("device", &device).Msg("count subscriptions finished")

	return device.toModel(), nil
}

func (s Repository) SaveDevice(ctx context.Context, device *model.Device) (int64, error) {
	logger := log.Ctx(ctx)
	dbctx := db.MustCtx(ctx)

	logger.Debug().Object("device", device).Msg("save device")

	if device.UpdatedAt.IsZero() {
		device.UpdatedAt = time.Now().UTC()
	}

	if device.ID == 0 {
		logger.Debug().Object("device", device).Msg("insert device")

		now := time.Now().UTC()

		var id int64

		err := dbctx.GetContext(ctx, &id, `
			INSERT INTO devices (user_id, name, dev_type, caption, updated_at, created_at)
			VALUES($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			device.User.ID, device.Name, device.DevType, device.Caption, now, now)
		if err != nil {
			return 0, aerr.Wrapf(err, "insert device failed")
		}

		return id, nil
	}

	// update
	logger.Debug().Object("device", device).Msg("update device")

	_, err := dbctx.ExecContext(ctx,
		"UPDATE devices SET dev_type=$1, caption=$2, updated_at=$3 WHERE id=$4",
		device.DevType, device.Caption, time.Now().UTC(), device.ID)
	if err != nil {
		return device.ID, aerr.Wrapf(err, "update device failed").WithMeta("device_id", device.ID)
	}

	return device.ID, nil
}

func (s Repository) ListDevices(ctx context.Context, userid int64) ([]model.Device, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("user_id", userid).Msg("list devices - count subscriptions")

	// all device have the same number of subscriptions
	var subscriptions int

	dbctx := db.MustCtx(ctx)

	err := dbctx.GetContext(ctx, &subscriptions,
		"SELECT count(*) FROM podcasts where user_id=$1 and subscribed",
		userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "count subscriptions error").WithMeta("user_id", userid)
	}

	logger.Debug().Int64("user_id", userid).Msg("list devices")

	devices := []DeviceDB{}

	err = dbctx.SelectContext(ctx, &devices, `
			SELECT d.id, d.user_id, d.name, d.dev_type, d.caption, $1 AS subscriptions,
				d.created_at, d.updated_at,
				u.id AS "user.id", u.name AS "user.name", u.username AS "user.username"
			FROM devices d
			JOIN users u ON u.id = d.user_id
			WHERE user_id=$2
			ORDER BY d.name`,
		subscriptions, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "select device failed").WithMeta("user_id", userid)
	}

	return devicesFromDb(devices), nil
}

func (s Repository) DeleteDevice(ctx context.Context, deviceid int64) error {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("device_id", deviceid).Msg("delete device")

	dbctx := db.MustCtx(ctx)

	_, err := dbctx.ExecContext(ctx, "UPDATE episodes SET device_id=NULL WHERE device_id=$1", deviceid)
	if err != nil {
		return aerr.Wrapf(err, "delete device failed")
	}

	_, err = dbctx.ExecContext(ctx, "DELETE FROM devices where id=$2", deviceid)
	if err != nil {
		return aerr.Wrapf(err, "delete device failed")
	}

	return nil
}
