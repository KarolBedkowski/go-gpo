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
	"maps"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type queryer interface {
	sqlx.QueryerContext
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
}

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

func (r *Repository) GetUser(ctx context.Context, username string) (*UserDB, error) {
	user := &UserDB{}

	err := r.db.QueryRowxContext(ctx,
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

func (r *Repository) GetDevice(ctx context.Context, userid int64, devicename string) (*DeviceDB, error) {
	return r.getDevice(ctx, r.db, userid, devicename)
}

func (r *Repository) getDevice(
	ctx context.Context,
	tx queryer,
	userid int64,
	devicename string,
) (*DeviceDB, error) {
	device := &DeviceDB{}
	err := tx.QueryRowxContext(ctx,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=? and name=?", userid, devicename).
		StructScan(device)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("query device error: %w", err)
	}

	err = tx.GetContext(
		ctx,
		&device.Subscriptions,
		"SELECT count(*) FROM podcasts where user_id=? and subscribed",
		userid,
	)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions error: %w", err)
	}

	return device, nil
}

func (r *Repository) getUserDevices(ctx context.Context, tx queryer, userid int64) (DevicesDB, error) {
	res := []*DeviceDB{}

	err := tx.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userid)
	if err != nil {
		return nil, fmt.Errorf("query device error: %w", err)
	}

	// all device have the same number of subscriptions

	var subscriptions int
	err = tx.GetContext(ctx, &subscriptions, "SELECT count(*) FROM podcasts where user_id=? and subscribed", userid)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions error: %w", err)
	}

	for _, r := range res {
		r.Subscriptions = subscriptions
	}

	return res, nil
}

func (r *Repository) SaveDevice(ctx context.Context, device *DeviceDB) (int64, error) {
	return r.saveDevice(ctx, r.db, device)
}

func (r *Repository) saveDevice(ctx context.Context, tx sqlx.ExecerContext, device *DeviceDB) (int64, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Interface("device", device).Msg("update device")

	if device.ID == 0 {
		res, err := tx.ExecContext(ctx,
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
	_, err := tx.ExecContext(ctx,
		"UPDATE devices SET dev_type=?, caption=?, updated_at=current_timestamp WHERE id=?",
		device.DevType, device.Caption, device.ID)
	if err != nil {
		return device.ID, fmt.Errorf("update device error: %w", err)
	}

	return device.ID, nil
}

func (r *Repository) ListDevices(ctx context.Context, userid int64) (DevicesDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Msg("list devices")

	res := []*DeviceDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT id, user_id, name, dev_type, caption, created_at, updated_at "+
			"FROM devices WHERE user_id=?", userid)
	if err != nil {
		return nil, fmt.Errorf("query devices error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetSubscribedPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (PodcastsDB, error) {
	return r.getSubscribedPodcasts(ctx, r.db, userid, since)
}

func (r *Repository) getSubscribedPodcasts(
	ctx context.Context,
	db queryer,
	userid int64,
	since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Time("since", since).Msg("get podcasts")

	res := []*PodcastDB{}

	err := db.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ? and subscribed", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	logger.Debug().Msgf("get podcasts: %d", len(res))

	return res, nil
}

func (r *Repository) GetPodcasts(
	ctx context.Context,
	userid int64,
	since time.Time,
) (PodcastsDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Time("since", since).Msg("get podcasts")

	res := []*PodcastDB{}

	err := r.db.SelectContext(ctx, &res,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.updated_at > ?", userid, since)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetPodcast(ctx context.Context, userid int64, podcasturl string) (*PodcastDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Str("podcasturl", podcasturl).Msg("get podcast")

	podcast := &PodcastDB{}
	err := r.db.QueryRowxContext(ctx,
		"SELECT p.id, p.user_id, p.url, p.title, p.subscribed, p.created_at, p.updated_at "+
			"FROM podcasts p "+
			"WHERE p.user_id=? AND p.url = ?", userid, podcasturl).
		StructScan(podcast)

	switch {
	case err == nil:
		return podcast, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, fmt.Errorf("query podcast %q error: %w", podcasturl, err)
	}
}

func (r *Repository) SavePodcast(ctx context.Context, user, device string, podcast ...*PodcastDB) error {
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

func (r *Repository) savePodcast(ctx context.Context, tx sqlx.ExecerContext, pod *PodcastDB) (int64, error) {
	if pod.UpdatedAt.IsZero() {
		pod.UpdatedAt = time.Now()
	}

	if pod.ID == 0 {
		if pod.CreatedAt.IsZero() {
			pod.CreatedAt = time.Now()
		}

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

func (r *Repository) GetEpisodes(
	ctx context.Context,
	userid, deviceid, podcastid int64,
	since time.Time,
	aggregated bool,
) ([]*EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Int64("podcastid", podcastid).
		Int64("deviceid", deviceid).Bool("aggregated", aggregated).
		Time("since", since).Msg("get podcasts")

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		"e.created_at, e.updated_at, p.url as podcast_url, d.name as device_name " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id JOIN devices d on d.id=e.device_id " +
		"WHERE p.user_id=? AND e.updated_at > ? ORDER BY e.updated_at"
	args := []any{userid, since}

	if deviceid > 0 {
		query += " AND e.device_id = ?"
		args = append(args, deviceid)
	}

	if podcastid > 0 {
		query += " AND e.podcast_id = ?"
		args = append(args, podcastid)
	}

	logger.Debug().Str("query", query).Interface("args", args).Msg("query")

	res := []*EpisodeDB{}
	err := r.db.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query episodes error: %w", err)
	}

	logger.Debug().Msgf("query result len=%d", len(res))

	if !aggregated {
		return res, nil
	}

	// TODO: refactor; load only last entries from db
	agr := make(map[int64]*EpisodeDB)
	for _, r := range res {
		agr[r.PodcastID] = r
	}

	return slices.Collect(maps.Values(agr)), nil
}

func (r *Repository) SaveEpisode(ctx context.Context, userid int64, episode ...*EpisodeDB) error {
	logger := log.Ctx(ctx)

	return r.inTransaction(ctx, func(tx *sqlx.Tx) error {
		podcasts, err := r.getSubscribedPodcasts(ctx, tx, userid, time.Time{})
		if err != nil {
			return err
		}

		podcastsmap := podcasts.ToIDsMap()

		devices, err := r.getUserDevices(ctx, tx, userid)
		if err != nil {
			return err
		}

		devicesmap := devices.ToIDsMap()

		for _, e := range episode {
			logger.Debug().Interface("episode", e).Msg("save episode")

			if pid, ok := podcastsmap[e.PodcastURL]; ok {
				// podcast already created
				e.PodcastID = pid
			} else {
				// insert podcast
				podcast := PodcastDB{UserID: userid, URL: e.PodcastURL, Subscribed: true}
				id, err := r.savePodcast(ctx, tx, &podcast)
				if err != nil {
					return fmt.Errorf("save new podcast %q error: %w", podcast.URL, err)
				}

				e.PodcastID = id
				podcastsmap[e.PodcastURL] = id
			}

			if did, ok := devicesmap[e.Device]; ok {
				e.DeviceID = did
			} else {
				// create device
				dev := DeviceDB{UserID: userid, Name: e.Device, DevType: "computer"}
				did, err := r.saveDevice(ctx, tx, &dev)
				if err != nil {
					return fmt.Errorf("save new device %q error: %w", e.Device, err)
				}

				e.DeviceID = did
				devicesmap[e.Device] = did
			}

			if err := r.saveEpisode(ctx, tx, e); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) saveEpisode(ctx context.Context, tx sqlx.ExecerContext, episode *EpisodeDB) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO episodes (podcast_id, device_id, title, url, action, started, position, total, "+
			"created_at, updated_at) "+
			"VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		episode.PodcastID,
		episode.DeviceID,
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
	return nil
}
