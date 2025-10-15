//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package model

import "time"

const (
	ActionUnsubscribe = "unsubscribe"
	ActionSubscribe   = "subscribe"
)

type Subscription struct {
	ID        int
	DeviceID  int `db:"device_id"`
	Podcast   string
	Action    string
	Timestamp time.Time `db:"ts"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewSubscription(deviceID int, podcast, action string) *Subscription {
	now := time.Now()
	return &Subscription{
		ID:        0,
		DeviceID:  deviceID,
		Podcast:   podcast,
		Action:    action,
		Timestamp: now,
	}
}

func (s *Subscription) NewAction(action string) *Subscription {
	now := time.Now()
	return &Subscription{
		ID:        0,
		DeviceID:  s.DeviceID,
		Podcast:   s.Podcast,
		Action:    s.Action,
		Timestamp: now,
	}
}
