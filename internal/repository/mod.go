//
// mod.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpodder/internal/model"
)

type Repository struct {
	db *sqlx.DB
}

func (r *Repository) Connect(driver, connstr string) error {
	var err error

	r.db, err = sqlx.Open(driver, connstr)
	if err != nil {
		return fmt.Errorf("open database error: %w", err)
	}

	if err := r.db.Ping(); err != nil {
		return fmt.Errorf("ping database error: %w", err)
	}

	return nil
}

func (r *Repository) GetUser(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.
		QueryRowxContext(ctx,
			"SELECT id, username, password, email, name, created_at, updated_at "+
				"FROM users WHERE username=?",
			username).
		StructScan(user)
	switch {
	case err == nil:
		return user, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, fmt.Errorf("get user error: %w", err)
	}
}

func (r *Repository) GetDevice(ctx context.Context, userID int, deviceID string) (*model.Device, error) {
	device := &model.Device{}
	err := r.db.
		QueryRowxContext(ctx,
			"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
				"FROM devices WHERE user_id=? and name=?", userID, deviceID).
		StructScan(device)

	switch {
	case err == nil:
		return device, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, fmt.Errorf("query device error: %w", err)
	}
}

func (r *Repository) SaveDevice(ctx context.Context, device *model.Device) (int, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("device", device).Msg("update device")

	if device.ID == 0 {
		res, err := r.db.
			ExecContext(ctx,
				"INSERT INTO devices (user_id, name, dev_type, caption) VALUES(?, ?, ?, ?)",
				device.UserID, device.Name, device.DevType, device.Caption)
		if err != nil {
			return 0, fmt.Errorf("insert new device error: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id error: %w", err)
		}

		return int(id), nil
	}

	// update
	_, err := r.db.
		ExecContext(ctx,
			"UPDATE devices SET dev_type=?, caption=?, updated_at=current_timestamp WHERE id=?",
			device.DevType, device.Caption, device.ID)
	if err != nil {
		return device.ID, fmt.Errorf("update device error: %w", err)
	}

	return device.ID, nil
}

func (r *Repository) ListDevices(ctx context.Context, userID int) ([]model.DeviceInfo, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	res := []model.DeviceInfo{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT name, caption, dev_type,  "+
			"  (SELECT count(*) from subscriptions s WHERE s.device_id = d.id where action='SUBSCRIBE') "+
			" - (SELECT count(*) from subscriptions s WHERE s.device_id = d.id where action='UNSUBSCRIBE') "+
			" as subscriptions "+
			"FROM devices d where user_id=?",
		userID)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetSubscriptions(ctx context.Context, deviceID int, since time.Time) ([]model.Subscription, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int("deviceID", deviceID).Time("sice", since).Msg("get subscriptions")

	res := []model.Subscription{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT id, device_id, podcast, action, ts, created_at, updated_at "+
			"FROM subscriptions WHERE device_id=? and updated_at > ?", deviceID, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetUserSubscriptions(ctx context.Context, userID int, since time.Time) ([]string, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int("userID", userID).Time("since", since).Msg("get user subscriptions")

	var res []string

	err := r.db.SelectContext(ctx, &res,
		"SELECT distinct podcast"+
			"FROM subscriptions s JOIN devices d on d.id = s.device_id "+
			"WHERE d.user_id=? ts > ?", userID, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) SaveSubscription(ctx context.Context, sub ...*model.Subscription) error {
	logger := log.Ctx(ctx)

	for _, s := range sub {
		logger.Debug().Interface("sub", s).Msg("update subscription")

		if s.ID == 0 {
			_, err := r.db.
				ExecContext(ctx,
					"INSERT INTO subscriptions (device_id, podcast, action, ts) VALUES(?, ?, ?, ?)",
					s.DeviceID, s.Podcast, s.Action, s.Timestamp)
			if err != nil {
				logger.Debug().Interface("sub", s).Err(err).Msg("insert subscription error")
				return fmt.Errorf("insert new subscription error: %w", err)
			}
		}

		// update
		_, err := r.db.
			ExecContext(ctx,
				"UPDATE subscriptions "+
					"SET podcast=?, action=?, ts=?, updated_at=current_timestamp "+
					"WHERE id=?",
				s.Podcast, s.Action, s.Timestamp, s.ID)
		if err != nil {
			logger.Debug().Interface("sub", s).Err(err).Msg("update subscription error")
			return fmt.Errorf("update subscriptions %d error: %w", s.ID, err)
		}
	}

	return nil
}
