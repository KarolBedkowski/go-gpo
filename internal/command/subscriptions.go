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
	"gitlab.com/kabes/go-gpo/internal/model"
)

var ErrEmptyUsername = aerr.NewSimple("username can't be empty").WithTag(aerr.ValidationError)

type ChangeSubscriptionsCmd struct {
	Username   string
	Devicename string
	Add        []string
	Remove     []string
	Timestamp  time.Time
}

func (s *ChangeSubscriptionsCmd) Sanitize() [][]string {
	var chAdd, chRem [][]string

	s.Add, chAdd = model.SanitizeURLs(s.Add)
	s.Remove, chRem = model.SanitizeURLs(s.Remove)

	changes := make([][]string, 0)
	changes = append(changes, chAdd...)
	changes = append(changes, chRem...)

	return changes
}

func (s *ChangeSubscriptionsCmd) Validate() error {
	if s.Username == "" {
		return ErrEmptyUsername
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
	Username      string
	Devicename    string
	Subscriptions []string
	Timestamp     time.Time
}

func (r *ReplaceSubscriptionsCmd) Validate() error {
	if r.Username == "" {
		return ErrEmptyUsername
	}

	return nil
}
