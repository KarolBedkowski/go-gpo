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
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

type GetEpisodesQuery struct {
	Since      time.Time
	UserName   string
	DeviceName string
	Podcast    string
	Limit      uint
	Aggregated bool
}

func (q *GetEpisodesQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
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

type GetEpisodesByPodcastQuery struct {
	Since      time.Time
	UserName   string
	PodcastID  int32
	Limit      uint
	Aggregated bool
}

func (q *GetEpisodesByPodcastQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	if q.PodcastID < 0 {
		return aerr.ErrValidation.WithMsg("invalid podcast id")
	}

	return nil
}

func (q *GetEpisodesByPodcastQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Int32("podcast_id", q.PodcastID).
		Time("since", q.Since).
		Bool("aggregate", q.Aggregated).
		Uint("limit", q.Limit)
}

//------------------------------------------------------------------------------

type GetEpisodeUpdatesQuery struct {
	Since          time.Time
	UserName       string
	DeviceName     string
	IncludeActions bool
}

func (q *GetEpisodeUpdatesQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
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
type GetLastEpisodesActionsQuery struct {
	Since    time.Time
	UserName string
	Limit    uint
}

func (q *GetLastEpisodesActionsQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	return nil
}

func (q *GetLastEpisodesActionsQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Time("since", q.Since).
		Uint("limit", q.Limit)
}
