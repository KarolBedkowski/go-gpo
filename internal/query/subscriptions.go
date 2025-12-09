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
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type GetUserSubscriptionsQuery struct {
	Since    time.Time
	UserName string
}

func (q *GetUserSubscriptionsQuery) Validate() error {
	if !validators.IsValidUserName(q.UserName) {
		return aerr.ErrValidation.WithUserMsg("invalid username")
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
	if !validators.IsValidUserName(q.UserName) {
		return aerr.ErrValidation.WithUserMsg("invalid username")
	}

	if !validators.IsValidDevName(q.DeviceName) {
		return aerr.ErrValidation.WithUserMsg("invalid username")
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
	if !validators.IsValidUserName(q.UserName) {
		return aerr.ErrValidation.WithUserMsg("invalid username")
	}

	return nil
}

func (q *GetSubscriptionChangesQuery) MarshalZerologObject(event *zerolog.Event) {
	event.Str("username", q.UserName).
		Str("devicename", q.DeviceName).
		Time("since", q.Since)
}
