package query

//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//
import (
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpo/internal/aerr"
)

type GetUserSubscriptionsQuery struct {
	Since    time.Time
	UserName string
}

func (q *GetUserSubscriptionsQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	return nil
}

func (q *GetUserSubscriptionsQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Time("since", q.Since)
}

//------------------------------------------------------------------------------

type GetSubscriptionsQuery struct {
	Since      time.Time
	UserName   string
	DeviceName string
}

func (q *GetSubscriptionsQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	if q.DeviceName == "" {
		return aerr.ErrValidation.WithMsg("device can't be empty")
	}

	return nil
}

func (q *GetSubscriptionsQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Str("devicename", q.DeviceName).
		Time("since", q.Since)
}

//------------------------------------------------------------------------------

type GetSubscriptionChangesQuery struct {
	Since      time.Time
	UserName   string
	DeviceName string
}

func (q *GetSubscriptionChangesQuery) Validate() error {
	if q.UserName == "" {
		return aerr.ErrValidation.WithMsg("user name can't be empty")
	}

	return nil
}

func (q *GetSubscriptionChangesQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Str("devicename", q.DeviceName).
		Time("since", q.Since)
}
