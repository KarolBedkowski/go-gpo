package repository

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
)

func (t *Transaction) GetEpisodes(
	ctx context.Context,
	userid, deviceid, podcastid int64,
	since time.Time,
	aggregated bool,
) ([]EpisodeDB, error) {
	logger := log.Ctx(ctx)
	logger.Debug().Int64("userid", userid).Int64("podcastid", podcastid).
		Int64("deviceid", deviceid).Bool("aggregated", aggregated).
		Time("since", since).Msg("get podcasts")

	query := "SELECT e.id, e.podcast_id, e.url, e.title, e.action, e.started, e.position, e.total, " +
		"e.created_at, e.updated_at, p.url as podcast_url, p.title as podcast_title, d.name as device_name " +
		"FROM episodes e JOIN podcasts p on p.id = e.podcast_id JOIN devices d on d.id=e.device_id " +
		"WHERE p.user_id=? AND e.updated_at > ? ORDER BY e.updated_at"
	args := []any{userid, since}

	if deviceid > 0 {
		query += " AND e.device_id = ?"
		args = append(args, deviceid) //nolint:wsl_v5
	}

	if podcastid > 0 {
		query += " AND e.podcast_id = ?"
		args = append(args, podcastid) //nolint:wsl_v5
	}

	logger.Debug().Str("query", query).Interface("args", args).Msg("query")

	res := []EpisodeDB{}

	err := t.tx.SelectContext(ctx, &res, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query episodes error: %w", err)
	}

	logger.Debug().Msgf("query result len=%d", len(res))

	if !aggregated {
		return res, nil
	}

	// TODO: refactor; load only last entries from db
	agr := make(map[int64]EpisodeDB)
	for _, t := range res {
		agr[t.PodcastID] = t
	}

	return slices.Collect(maps.Values(agr)), nil
}

func (t *Transaction) SaveEpisode(ctx context.Context, userid int64, episode ...EpisodeDB) error {
	logger := log.Ctx(ctx)

	podcasts, err := t.GetSubscribedPodcasts(ctx, userid, time.Time{})
	if err != nil {
		return err
	}

	podcastsmap := podcasts.ToIDsMap()

	devices, err := t.getUserDevices(ctx, userid)
	if err != nil {
		return err
	}

	devicesmap := devices.ToIDsMap()

	for _, eps := range episode {
		logger.Debug().Interface("episode", eps).Msg("save episode")

		if pid, ok := podcastsmap[eps.PodcastURL]; ok {
			// podcast already created
			eps.PodcastID = pid
		} else {
			// insert podcast
			id, err := t.createNewPodcast(ctx, userid, eps.PodcastURL)
			if err != nil {
				return fmt.Errorf("save new podcast %q error: %w", eps.PodcastURL, err)
			}

			eps.PodcastID = id
			podcastsmap[eps.PodcastURL] = id
		}

		if did, ok := devicesmap[eps.Device]; ok {
			eps.DeviceID = did
		} else {
			// create device
			did, err := t.createNewDevice(ctx, userid, eps.Device)
			if err != nil {
				return fmt.Errorf("save new device %q error: %w", eps.Device, err)
			}

			eps.DeviceID = did
			devicesmap[eps.Device] = did
		}

		if err := t.saveEpisode(ctx, eps); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transaction) saveEpisode(ctx context.Context, episode EpisodeDB) error {
	_, err := t.tx.ExecContext(
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
