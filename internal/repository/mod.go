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

	if err := r.db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("ping database error: %w", err)
	}

	return nil
}

func (r *Repository) GetUser(ctx context.Context, username string) (*model.UserDB, error) {
	user := &model.UserDB{}

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

func (r *Repository) GetDevice(ctx context.Context, userID int, deviceID string) (*model.DeviceDB, error) {
	device := &model.DeviceDB{}
	err := r.db.
		QueryRowxContext(ctx,
			"SELECT id, user_id, name, dev_type, caption, subscriptions, created_at, updated_at "+
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

func (r *Repository) SaveDevice(ctx context.Context, device *model.DeviceDB) (int, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("device", device).Msg("update device")

	if device.ID == 0 {
		res, err := r.db.
			ExecContext(ctx,
				"INSERT INTO devices (user_id, name, dev_type, caption, subscriptions) VALUES(?, ?, ?, ?, ?)",
				device.UserID, device.Name, device.DevType, device.Caption, device.Subscriptions)
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
			"UPDATE devices SET dev_type=?, caption=?, subscriptions=?, updated_at=current_timestamp WHERE id=?",
			device.DevType, device.Caption, device.Subscriptions, device.ID)
	if err != nil {
		return device.ID, fmt.Errorf("update device error: %w", err)
	}

	return device.ID, nil
}

func (r *Repository) ListDevices(ctx context.Context, userID int) ([]model.DeviceDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	res := []model.DeviceDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, subscriptions, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userID)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetSubscriptionChanges(
	ctx context.Context,
	deviceID int,
	since time.Time,
) ([]*model.SubscriptionDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int("deviceID", deviceID).Time("sice", since).Msg("get subscriptions")

	res := []*model.SubscriptionChangenDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT s.id, s.device_id, s.podcast_id, p.url as podcast_url, s.action, s.created_at, s.updated_at "+
			"FROM subscriptions s "+
			"JOIN podcasts p on p.id = s.podcast_id "+
			"WHERE s.device_id=? and s.updated_at > ?", deviceID, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetSubscriptions(
	ctx context.Context,
	deviceID int,
	since time.Time,
) (model.SubscribedPodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int("deviceID", deviceID).Time("sice", since).Msg("get subscriptions")

	res := []*model.SubscribedPodcastDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT s.id as subscription_id, p.id as podcast_id, p.url as podcast_url "+
			"FROM subscriptions s "+
			"JOIN podcasts p on p.id = s.podcast_id "+
			"WHERE s.device_id=? and s.updated_at > ?", deviceID, since)
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
		"SELECT distinct p.url "+
			"FROM subscriptions s "+
			"JOIN podcasts p on p.id = s.podcast_id "+
			"JOIN devices d ON d.id = s.device_id "+
			"WHERE d.user_id=? AND updated_at > ?", userID, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) SaveSubscription(ctx context.Context, subscription ...*model.SubscriptionDB) error {
	logger := log.Ctx(ctx)
	for _, sub := range subscription {
		logger.Debug().Interface("sub", sub).Msg("update subscription")

		if sub.ID == 0 {
			_, err := r.db.
				ExecContext(
					ctx,
					"INSERT INTO subscriptions (device_id, podcast_id, action, created_at, updated_at) "+
						"VALUES(?, ?, ?, ?, ?)",
					sub.DeviceID,
					sub.PodcastID,
					sub.Action,
					sub.CreatedAt,
					sub.UpdatedAt,
				)
			if err != nil {
				logger.Debug().Interface("sub", sub).Err(err).Msg("insert subscription error")

				return fmt.Errorf("insert new subscription error: %w", err)
			}
		}

		// update
		_, err := r.db.
			ExecContext(ctx,
				"UPDATE subscriptions SET podcast_id=?, action=?, updated_at=? WHERE id=?",
				sub.PodcastID, sub.Action, sub.UpdatedAt, sub.ID)
		if err != nil {
			logger.Debug().Interface("sub", sub).Err(err).Msg("update subscription error")

			return fmt.Errorf("update subscriptions %d error: %w", sub.ID, err)
		}
	}

	return nil
}

func (r *Repository) GetOrCreatePodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
	podcast, err := r.GetPodcast(ctx, userid, url)
	if err != nil {
		return nil, fmt.Errorf("get or create podcast error: %w", err)
	}

	if podcast != nil {
		return podcast, nil
	}

	return r.InsertPodcast(ctx, userid, url)
}

func (r *Repository) GetPodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
	podcast := &model.PodcastDB{}
	err := r.db.
		QueryRowxContext(ctx,
			"SELECT id, user_id, title, url, created_at, updated_at FROM podcasts WHERE user_id=? and url=?",
			userid, url).
		StructScan(podcast)

	switch {
	case err == nil:
		return podcast, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, fmt.Errorf("query device error: %w", err)
	}
}

func (r *Repository) InsertPodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
	_, err := r.db.ExecContext(ctx, "INSERT INTO podcasts (user_id, title, url) VALUE(?, ?, ?)", userid, url, url)
	if err != nil {
		return nil, fmt.Errorf("insert podcast for user=%d, url=%q error: %w", userid, url, err)
	}

	return r.GetPodcast(ctx, userid, url)
}
