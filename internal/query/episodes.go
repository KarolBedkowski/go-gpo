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
	UserName   string
	DeviceName string
	Podcast    string
	Since      time.Time
	Aggregated bool
	Limit      uint
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

type GetEpisodeUpdatesQuery struct {
	UserName       string
	DeviceName     string
	Since          time.Time
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
