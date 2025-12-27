package query

//
// episodes.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

// GetEpisodesQuery define arguments used to get episodes.
type GetEpisodesQuery struct {
	Since      time.Time
	UserName   string
	DeviceName string
	Podcast    string
	Limit      uint
	Aggregated bool
}

func (q *GetEpisodesQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if q.DeviceName != "" && !validators.IsValidDevName(q.DeviceName) {
		return common.ErrInvalidDevice.WithUserMsg("invalid device name")
	}

	return nil
}

func (q *GetEpisodesQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Str("podcast", q.Podcast).
		Str("device", q.DeviceName).
		Time("since", q.Since).
		Bool("aggregate", q.Aggregated).
		Uint("limit", q.Limit)
}

//------------------------------------------------------------------------------

// GetEpisodesByPodcastQuery define arguments to get episodes for specified podcast.
type GetEpisodesByPodcastQuery struct {
	Since      time.Time
	UserName   string
	PodcastID  int64
	Limit      uint
	Aggregated bool
}

func (q *GetEpisodesByPodcastQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if q.PodcastID < 0 {
		return common.ErrInvalidPodcast.WithUserMsg("invalid podcast id")
	}

	return nil
}

func (q *GetEpisodesByPodcastQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Int64("podcast_id", q.PodcastID).
		Time("since", q.Since).
		Bool("aggregate", q.Aggregated).
		Uint("limit", q.Limit)
}

//------------------------------------------------------------------------------

// GetEpisodeUpdatesQuery is used to get episode updates for user/device.
type GetEpisodeUpdatesQuery struct {
	Since          time.Time
	UserName       string
	DeviceName     string
	IncludeActions bool
}

func (q *GetEpisodeUpdatesQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if q.DeviceName != "" && !validators.IsValidDevName(q.DeviceName) {
		return common.ErrInvalidDevice.WithUserMsg("invalid device name")
	}

	return nil
}

func (q *GetEpisodeUpdatesQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Str("device", q.DeviceName).
		Time("since", q.Since).
		Bool("include_actions", q.IncludeActions)
}

// ------------------------------------------------------------------------------

// GetLastEpisodesActionsQuery get last episodes actions.
type GetLastEpisodesActionsQuery struct {
	Since    time.Time
	UserName string
	Limit    uint
}

func (q *GetLastEpisodesActionsQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	return nil
}

func (q *GetLastEpisodesActionsQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Time("since", q.Since).
		Uint("limit", q.Limit)
}
