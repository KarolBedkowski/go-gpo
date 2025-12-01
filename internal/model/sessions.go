package model

//
// sessions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"time"

	"github.com/rs/zerolog"
)

type Session struct {
	CreatedAt time.Time
	Data      map[any]any
	SID       string
}

func (s *Session) IsValid(maxlifetime time.Duration) bool {
	return !s.CreatedAt.Add(maxlifetime).Before(time.Now().UTC())
}

func (s *Session) MarshalZerologObject(event *zerolog.Event) {
	event.Str("sid", s.SID).
		Any("Data", s.Data).
		Time("created_at", s.CreatedAt).
		Dur("age", time.Since(s.CreatedAt))
}
