package model

import "time"

//
// sessions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

type Session struct {
	SID       string
	Data      []byte
	CreatedAt time.Time
}
