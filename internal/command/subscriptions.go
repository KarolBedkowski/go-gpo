package command

//
// subscriptions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

import (
	"slices"
	"time"

	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/validators"
)

type ChangeSubscriptionsCmd struct {
	Timestamp  time.Time
	UserName   string
	DeviceName string
	Add        []string
	Remove     []string
}

func (s *ChangeSubscriptionsCmd) Sanitize() [][]string {
	var chAdd, chRem [][]string

	s.Add, chAdd = validators.SanitizeURLs(s.Add)
	s.Remove, chRem = validators.SanitizeURLs(s.Remove)

	changes := make([][]string, 0)
	changes = append(changes, chAdd...)
	changes = append(changes, chRem...)

	return changes
}

func (s *ChangeSubscriptionsCmd) Validate() error {
	if !validators.IsValidUserName(s.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if !validators.IsValidDevName(s.DeviceName) {
		return common.ErrInvalidDevice.WithUserMsg("invalid device name")
	}

	for _, i := range s.Add {
		if slices.Contains(s.Remove, i) {
			return aerr.ErrValidation.WithUserMsg("duplicated url: %s", i)
		}
	}

	return nil
}

type ChangeSubscriptionsCmdResult struct {
	ChangedURLs [][]string
}

//---------------------------------------------------------------------

type ReplaceSubscriptionsCmd struct {
	Timestamp     time.Time
	UserName      string
	DeviceName    string
	Subscriptions []string
}

func (r *ReplaceSubscriptionsCmd) Validate() error {
	if !validators.IsValidUserName(r.UserName) {
		return common.ErrInvalidUser.WithUserMsg("invalid username")
	}

	if !validators.IsValidDevName(r.DeviceName) {
		return common.ErrInvalidDevice.WithUserMsg("invalid device name")
	}

	return nil
}
