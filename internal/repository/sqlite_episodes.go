package repository

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

func (s sqliteRepository) GetEpisode(
	ctx context.Context,
	dbctx DBContext,
	userid, podcastid int64,
	episodeURL string,
) (EpisodeDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_episodes").Logger()
	logger.Debug().Int64("user_id", userid).Int64("podcast_id", podcastid).Str("episode_urk", episodeURL).
		Msgf("get episode")

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id " +
		"WHERE p.user_id=? AND e.podcast_id = ? and e.url = ?"

	res := EpisodeDB{}

	err := dbctx.GetContext(ctx, &res, query, userid, podcastid, episodeURL)
	if err != nil {
		return res, aerr.Wrapf(err, "query episode failed").WithTag(aerr.InternalError)
	}

	return res, nil
}

// ListEpisodeActions for user, and optionally for device and podcastid.
// If deviceid is given, return actions from OTHER than given devices.
func (s sqliteRepository) ListEpisodeActions(
	ctx context.Context,
	dbctx DBContext,
	userid int64, deviceid, podcastid *int64,
	since time.Time,
	aggregated bool,
	lastelements int,
) ([]EpisodeDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_episodes").Logger()
	logger.Debug().Int64("user_id", userid).Any("podcast_id", podcastid).Any("device_id", deviceid).
		Msgf("get episodes since=%s aggregated=%v", since, aggregated)

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title, d.name as device_name " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id JOIN devices d on d.id=e.device_id " +
		"WHERE p.user_id=? AND e.updated_at > ?"
	args := []any{userid, since}

	if deviceid != nil {
		query += " AND e.device_id != ?"
		args = append(args, *deviceid) //nolint:wsl_v5
	}

	if podcastid != nil {
		query += " AND e.podcast_id = ?"
		args = append(args, *podcastid) //nolint:wsl_v5
	}

	query += " ORDER BY e.updated_at DESC"

	if lastelements > 0 {
		query += " LIMIT " + strconv.Itoa(lastelements)
	}

	logger.Debug().Msgf("get episodes - query=%q, args=%v", query, args)

	res := []EpisodeDB{}

	err := dbctx.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError).
			WithMeta("sql", query, "args", args)
	}

	if !aggregated {
		return res, nil
	}

	logger.Debug().Msgf("get episodes - aggregate %d episodes", len(res))

	// TODO: refactor; load only last entries for each podcast from db
	agr := make(map[int64]EpisodeDB)
	for _, t := range res {
		agr[t.PodcastID] = t
	}

	return slices.Collect(maps.Values(agr)), nil
}

func (s sqliteRepository) ListFavorites(ctx context.Context, dbctx DBContext, userid int64) ([]EpisodeDB, error) {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_episodes").Logger()
	logger.Debug().Int64("user_id", userid).Msg("get favorites")

	query := "SELECT e.id, e.podcast_id, e.url, e.title, " +
		" e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id " +
		"JOIN settings s on s.episode_id = e.id " +
		"WHERE p.user_id=? AND s.scope = 'episode' and s.key = 'is_favorite'"

	res := []EpisodeDB{}

	err := dbctx.SelectContext(ctx, &res, query, userid)
	if err != nil {
		return nil, aerr.Wrapf(err, "query episodes failed").WithTag(aerr.InternalError)
	}

	return res, nil
}

func (s sqliteRepository) SaveEpisode(ctx context.Context, dbctx DBContext, userid int64, episode ...EpisodeDB) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_episodes").Logger()
	logger.Debug().Int64("user_id", userid).Msgf("save episode")

	podcasts, err := s.ListSubscribedPodcasts(ctx, dbctx, userid, time.Time{})
	if err != nil {
		return err
	}

	// cache podcasts
	podcastsmap := podcasts.ToIDsMap()

	devices, err := s.ListDevices(ctx, dbctx, userid)
	if err != nil {
		return err
	}

	// cache devices
	devicesmap := devices.ToIDsMap()

	for _, eps := range episode {
		logger.Debug().Object("episode", eps).Msg("update episode")

		if pid, ok := podcastsmap[eps.PodcastURL]; ok {
			// podcast already created
			eps.PodcastID = pid
		} else {
			// insert podcast
			id, err := s.SavePodcast(ctx, dbctx, &PodcastDB{UserID: userid, URL: eps.PodcastURL, Subscribed: true})
			if err != nil {
				return aerr.Wrapf(err, "create new podcast failed")
			}

			eps.PodcastID = id
			podcastsmap[eps.PodcastURL] = id
		}

		if did, ok := devicesmap[eps.Device]; ok {
			eps.DeviceID = did
		} else {
			// create device
			did, err := s.SaveDevice(ctx, dbctx, &DeviceDB{UserID: userid, Name: eps.Device, DevType: "other"})
			if err != nil {
				return aerr.Wrapf(err, "create new device failed")
			}

			eps.DeviceID = did
			devicesmap[eps.Device] = did
		}

		if err := s.saveEpisode(ctx, dbctx, eps); err != nil {
			return err
		}
	}

	return nil
}

func (s sqliteRepository) saveEpisode(ctx context.Context, dbctx DBContext, episode EpisodeDB) error {
	logger := log.Ctx(ctx).With().Str("mod", "sqlite_repo_episodes").Logger()
	logger.Debug().Object("episode", episode).Msg("save episode")

	_, err := dbctx.ExecContext(
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
		return aerr.Wrapf(err, "insert episode failed").WithTag(aerr.InternalError).
			WithMeta("podcast_id", episode.PodcastID, "episode_url", episode.URL)
	}

	return nil
}
