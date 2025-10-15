// devices.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
package model

import "time"

type Device struct {
	ID        int
	UserID    int `db:"user_id"`
	Name      string
	DevType   string `db:"dev_type"`
	Caption   string
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type DeviceInfo struct {
	Name          string `json:"id"`
	DevType       string `db:"dev_type"`
	Caption       string
	Subscriptions int
}
