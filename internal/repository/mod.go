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

func (r *Repository) inTransaction(ctx context.Context, f func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx error: %w", err)
	}

	if err := f(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("%w; with rollback error: %w", err, rerr)
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx error: %w", err)
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

func (r *Repository) GetDevice(ctx context.Context, userID int64, deviceID string) (*model.DeviceDB, error) {
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

func (r *Repository) SaveDevice(ctx context.Context, device *model.DeviceDB) (int64, error) {
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

		return id, nil
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

func (r *Repository) ListDevices(ctx context.Context, userID int64) (model.DevicesDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	res := []*model.DeviceDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, subscriptions, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userID)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetSubscribedPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (model.PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userID", userid).Time("since", since).Msg("get podcasts")

	res := []*model.PodcastDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ? and p.subscribed", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (model.PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userID", userid).Time("since", since).Msg("get podcasts")

	res := []*model.PodcastDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ?", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

// func (r *Repository) GetSubscriptions(
// 	ctx context.Context,
// 	deviceID int,
// 	since time.Time,
// ) (model.SubscribedPodcastsDB, error) {
// 	logger := log.Ctx(ctx)
// 	logger.Debug().Int("deviceID", deviceID).Time("sice", since).Msg("get subscriptions")

// 	res := []*model.SubscribedPodcastDB{}

// 	err := r.db.SelectContext(ctx, &res,
// 		"SELECT s.id as subscription_id, p.id as podcast_id, p.url as podcast_url "+
// 			"FROM subscriptions s "+
// 			"JOIN podcasts p on p.id = s.podcast_id "+
// 			"WHERE s.device_id=? and s.updated_at > ?", deviceID, since)
// 	if err != nil {
// 		return nil, fmt.Errorf("query subscriptions error: %w", err)
// 	}

// 	return res, nil
// }

// func (r *Repository) GetUserSubscriptions(ctx context.Context, userID int, since time.Time) ([]string, error) {
// 	logger := log.Ctx(ctx)
// 	logger.Debug().Int("userID", userID).Time("since", since).Msg("get user subscriptions")

// 	var res []string

// 	err := r.db.SelectContext(ctx, &res,
// 		"SELECT distinct p.url "+
// 			"FROM subscriptions s "+
// 			"JOIN podcasts p on p.id = s.podcast_id "+
// 			"JOIN devices d ON d.id = s.device_id "+
// 			"WHERE d.user_id=? AND updated_at > ?", userID, since)
// 	if err != nil {
// 		return nil, fmt.Errorf("query subscriptions error: %w", err)
// 	}

// 	return res, nil
// }

func (r *Repository) SavePodcast(ctx context.Context, user, device string, podcast ...*model.PodcastDB) error {
	logger := log.Ctx(ctx)

	return r.inTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, pod := range podcast {
			logger.Debug().Interface("podcast", pod).Msg("save podcast")
			if _, err := r.savePodcast(ctx, tx, pod); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) savePodcast(ctx context.Context, tx *sqlx.Tx, pod *model.PodcastDB) (int64, error) {
	if pod.ID == 0 {
		res, err := tx.ExecContext(
			ctx,
			"INSERT INTO podcasts (user_id, title, url, subscribed, created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?)",
			pod.UserID,
			pod.Title,
			pod.URL,
			pod.Subscribed,
			pod.CreatedAt,
			pod.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("insert new podcast %q error: %w", pod.URL, err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last id for %q error: %w", pod.URL, err)
		}

		return id, nil
	}

	// update
	_, err := tx.ExecContext(ctx,
		"UPDATE podcasts SET subscribed=?, title=?, url=?, updated_at=? WHERE id=?",
		pod.Subscribed, pod.Title, pod.URL, pod.UpdatedAt, pod.ID)
	if err != nil {
		return 0, fmt.Errorf("update subscriptions %d error: %w", pod.ID, err)
	}

	return pod.ID, nil
}

func (r *Repository) getOrCreateDevice(ctx context.Context, tx *sqlx.Tx, userid int64, devicename string) (*model.DeviceDB, error) {
	dev, err := r.getDevice(ctx, tx, userid, devicename)
	if dev != nil {
		return dev, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query device error: %w", err)
	}

	// insert new device
	dev = &model.DeviceDB{
		UserID:  userid,
		Name:    devicename,
		DevType: "",
	}

	if err := r.saveDevice(ctx, tx, dev); err != nil {
		return nil, fmt.Errorf("create device error: %w", err)
	}

	// get dev

	dev, err = r.getDevice(ctx, tx, userid, devicename)
	if dev != nil {
		return dev, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query device error: %w", err)
	}

	return dev, nil
}

func (r *Repository) getDevice(ctx context.Context, tx *sqlx.Tx, userid int64, devicename string) (*model.DeviceDB, error) {
	device := &model.DeviceDB{}
	err := tx.
		QueryRowxContext(ctx,
			"SELECT id, user_id, name, dev_type, caption, subscriptions, created_at, updated_at "+
				"FROM devices WHERE user_id=? AND name=?", userid, devicename).
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

func (r *Repository) saveDevice(ctx context.Context, tx *sqlx.Tx, device *model.DeviceDB) error {
	if device.ID == 0 {
		_, err := tx.
			ExecContext(ctx,
				"INSERT INTO devices (user_id, name, dev_type, caption, subscriptions) VALUES(?, ?, ?, ?, ?)",
				device.UserID, device.Name, device.DevType, device.Caption, device.Subscriptions)
		if err != nil {
			return fmt.Errorf("insert new device error: %w", err)
		}

		return nil
	}

	// update
	_, err := tx.
		ExecContext(ctx,
			"UPDATE devices SET dev_type=?, caption=?, subscriptions=?, updated_at=current_timestamp WHERE id=?",
			device.DevType, device.Caption, device.Subscriptions, device.ID)
	if err != nil {
		return fmt.Errorf("update device error: %w", err)
	}

	return nil
}

func (r *Repository) GetEpisodes(ctx context.Context, userid, podcastid int64, since time.Time) ([]*model.EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Int64("podcastid", podcastid).Time("since", since).Msg("get podcasts")

	res := []*model.EpisodeDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, e.created_at, e.updated_at, "+
			"p.url as podcast_url "+
			"FROM episodes e JOIN podcasts p on p.id = e.podcast_id "+
			"WHERE p.user_id=? AND p.updated_at > ?", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query episodes error: %w", err)
	}

	return res, nil
}

func (r *Repository) SaveEpisode(ctx context.Context, episode ...*model.EpisodeDB) error {
	logger := log.Ctx(ctx)

	return r.inTransaction(ctx, func(tx *sqlx.Tx) error {
		newpodcasts := make(map[string]int64)

		for _, e := range episode {
			logger.Debug().Interface("episode", e).Msg("save episode")

			if e.PodcastID == 0 {
				// add podcast
				if pid, ok := newpodcasts[e.Podcast.URL]; ok {
					// podcast already created
					e.PodcastID = pid
				} else {
					// insert podcast
					id, err := r.savePodcast(ctx, tx, e.Podcast)
					if err != nil {
						return fmt.Errorf("save new podcast %q error: %w", e.Podcast.URL, err)
					}

					e.PodcastID = id
					newpodcasts[e.Podcast.URL] = id
				}
			}

			if err := r.saveEpisode(ctx, tx, e); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) saveEpisode(ctx context.Context, tx *sqlx.Tx, episode *model.EpisodeDB) error {
	if episode.ID == 0 {
		_, err := tx.ExecContext(
			ctx,
			"INSERT INTO episodes (podcast_id, title, url, action, started, position, total, "+
				"created_at, updated_at) "+
				"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			episode.PodcastID,
			episode.Title,
			episode.URL,
			episode.Action,
			episode.Started,
			episode.Position,
			episode.Total,
			episode.CreatedAt,
			episode.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert new podcast %d episode %q error: %w", episode.PodcastID,
				episode.URL, err)
		}
	}

	// update
	_, err := tx.ExecContext(ctx,
		"UPDATE episodes SET title=?, url=?, action=?, started=?, position=?, total=?, updated_at=? "+
			"WHERE id=?",
		episode.Title,
		episode.URL,
		episode.Action,
		episode.Started,
		episode.Position,
		episode.Total,
		episode.UpdatedAt,
		episode.ID)
	if err != nil {
		return fmt.Errorf("update subscriptions %d error: %w", episode.ID, err)
	}

	return nil
}

// func (r *Repository) GetOrCreatePodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
// 	podcast, err := r.GetPodcast(ctx, userid, url)
// 	if err != nil {
// 		return nil, fmt.Errorf("get or create podcast error: %w", err)
// 	}

// 	if podcast != nil {
// 		return podcast, nil
// 	}

// 	return r.InsertPodcast(ctx, userid, url)
// }

// func (r *Repository) GetPodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
// 	podcast := &model.PodcastDB{}
// 	err := r.db.
// 		QueryRowxContext(ctx,
// 			"SELECT id, user_id, title, url, created_at, updated_at FROM podcasts WHERE user_id=? and url=?",
// 			userid, url).
// 		StructScan(podcast)

// 	switch {
// 	case err == nil:
// 		return podcast, nil
// 	case errors.Is(err, sql.ErrNoRows):
// 		return nil, nil
// 	default:
// 		return nil, fmt.Errorf("query device error: %w", err)
// 	}
// }

// func (r *Repository) InsertPodcast(ctx context.Context, userid int, url string) (*model.PodcastDB, error) {
// 	_, err := r.db.ExecContext(ctx, "INSERT INTO podcasts (user_id, title, url) VALUE(?, ?, ?)", userid, url, url)
// 	if err != nil {
// 		return nil, fmt.Errorf("insert podcast for user=%d, url=%q error: %w", userid, url, err)
// 	}

// 	return r.GetPodcast(ctx, userid, url)
// }
