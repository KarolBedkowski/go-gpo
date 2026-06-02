//
// device.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Distributed under terms of the GPLv3 license.
//

package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/command"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/query"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type SubscriptionsSrv struct {
	dbi          repository.Database
	podcastsRepo repository.Podcasts
	usersRepo    repository.Users
	devicesRepo  repository.Devices
}

func NewSubscriptionsSrv(i do.Injector) (*SubscriptionsSrv, error) {
	return &SubscriptionsSrv{
		dbi:          do.MustInvoke[repository.Database](i),
		podcastsRepo: do.MustInvoke[repository.Podcasts](i),
		usersRepo:    do.MustInvoke[repository.Users](i),
		devicesRepo:  do.MustInvoke[repository.Devices](i),
	}, nil
}

// GetUserSubscriptions is simple api.
func (s *SubscriptionsSrv) GetUserSubscriptions(ctx context.Context, query *query.GetUserSubscriptionsQuery,
) (model.Podcasts, error) {
	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validation query failed")
	}

	return s.getSubsctiptions(ctx, query.UserName, "", query.Since)
}

// GetSubscriptions is simple api.
func (s *SubscriptionsSrv) GetSubscriptions(ctx context.Context, query *query.GetSubscriptionsQuery,
) (model.Podcasts, error) {
	if err := query.Validate(); err != nil {
		return nil, aerr.Wrapf(err, "validation query failed")
	}

	return s.getSubsctiptions(ctx, query.UserName, query.DeviceName, query.Since)
}

// ReplaceSubscriptions replace all subscriptions for given user. Create device when no exists.
func (s *SubscriptionsSrv) ReplaceSubscriptions(
	ctx context.Context,
	cmd *command.ReplaceSubscriptionsCmd,
) error {
	if err := cmd.Validate(); err != nil {
		return aerr.Wrapf(err, "validate command failed")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.dbi, func(ctx context.Context) error {
		user, err := s.getUser(ctx, cmd.UserName)
		if err != nil {
			return err
		}

		// check dev
		_, err = s.getUserDevice(ctx, user.ID, cmd.DeviceName)
		if errors.Is(err, aerr.ErrNoData) {
			_, err = s.createUserDevice(ctx, user, cmd.DeviceName)
		}

		if err != nil {
			return err
		}

		// get all podcasts for user
		subscribed, err := s.podcastsRepo.ListPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return aerr.Wrapf(err, "list podcasts error")
		}

		for _, p := range prepareReplaceSubscriptions(subscribed, cmd) {
			p.User = user
			if _, err := s.podcastsRepo.SavePodcast(ctx, &p); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

func prepareReplaceSubscriptions(subscribed model.Podcasts, cmd *command.ReplaceSubscriptionsCmd) []model.Podcast {
	changes := make([]model.Podcast, 0, len(cmd.Subscriptions))
	// remove subscriptions found in db but not in currentSubs
	for _, sub := range subscribed {
		if sub.Subscribed && !slices.Contains(cmd.Subscriptions, sub.URL) {
			sub.SetUnsubscribed(cmd.Timestamp)
			changes = append(changes, sub)
		}
	}

	// add or set subscribed flag for podcast in currentSubs; update updated_at
	for _, sub := range cmd.Subscriptions {
		podcast, ok := subscribed.FindPodcastByURL(sub)
		if !ok {
			podcast = model.Podcast{URL: sub}
		} else if podcast.Subscribed {
			continue
		}

		if podcast.SetSubscribed(cmd.Timestamp) {
			changes = append(changes, podcast)
		}
	}

	return changes
}

func (s *SubscriptionsSrv) ChangeSubscriptions(
	ctx context.Context, cmd *command.ChangeSubscriptionsCmd,
) (command.ChangeSubscriptionsCmdResult, error) {
	res := command.ChangeSubscriptionsCmdResult{
		ChangedURLs: cmd.Sanitize(),
	}

	if err := cmd.Validate(); err != nil {
		return res, aerr.Wrapf(err, "validate command failed")
	}

	err := db.InTransaction(ctx, s.dbi, func(ctx context.Context) error {
		user, err := s.getUser(ctx, cmd.UserName)
		if err != nil {
			return err
		}

		// check device if given
		if cmd.DeviceName != "" {
			if _, err = s.getUserDevice(ctx, user.ID, cmd.DeviceName); err != nil {
				return err
			}
		}

		userpodcasts, err := s.podcastsRepo.ListPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return aerr.Wrapf(err, "list podcasts error")
		}

		for _, p := range preparePodcastChanges(userpodcasts, cmd) {
			p.User = user
			if _, err := s.podcastsRepo.SavePodcast(ctx, &p); err != nil {
				res.PodcastsModified = 0

				return aerr.Wrapf(err, "save podcast error")
			}

			res.PodcastsModified += 1
		}

		return nil
	})

	return res, err //nolint:wrapcheck
}

func preparePodcastChanges(userpodcasts model.Podcasts, cmd *command.ChangeSubscriptionsCmd) []model.Podcast {
	podchanges := make([]model.Podcast, 0, len(cmd.Add)+len(cmd.Remove))

	// removed
	for _, sub := range cmd.Remove {
		if podcast, ok := userpodcasts.FindPodcastByURL(sub); ok {
			if podcast.SetUnsubscribed(cmd.Timestamp) {
				podchanges = append(podchanges, podcast)
			}
		}
	}

	for _, sub := range cmd.Add {
		podcast, ok := userpodcasts.FindPodcastByURL(sub)
		if !ok { // new
			// user is set on save
			podcast = model.Podcast{URL: sub}
		} else if podcast.Subscribed {
			continue
		}

		if podcast.SetSubscribed(cmd.Timestamp) {
			podchanges = append(podchanges, podcast)
		}
	}

	return podchanges
}

func (s *SubscriptionsSrv) GetSubscriptionChanges(ctx context.Context, query *query.GetSubscriptionChangesQuery) (
	model.SubscriptionState, error,
) {
	if err := query.Validate(); err != nil {
		return model.SubscriptionState{}, aerr.Wrapf(err, "validation query failed")
	}

	podcasts, err := s.getPodcasts(ctx, query.UserName, query.DeviceName, query.Since)
	if err != nil {
		return model.SubscriptionState{}, err
	}

	common.TraceLazyPrintf(ctx, "GetSubscriptionChanges: podcasts loaded")

	state := model.SubscriptionState{
		Added:   make([]model.Podcast, 0, len(podcasts)),
		Removed: make([]model.Podcast, 0),
	}

	for _, p := range podcasts {
		if p.Subscribed {
			state.Added = append(state.Added, p)
		} else {
			state.Removed = append(state.Removed, p)
		}
	}

	return state, nil
}

// ------------------------------------------------------

func (s *SubscriptionsSrv) getSubsctiptions(ctx context.Context, username, devicename string, since time.Time,
) (model.Podcasts, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.dbi, func(ctx context.Context) (model.Podcasts, error) {
		user, err := s.usersRepo.GetUser(ctx, username)
		if err != nil {
			return nil, aerr.Wrapf(err, "get user error")
		}

		if devicename != "" {
			// validate is device exists when device name is given and mark is seen.
			_, err := s.getUserDevice(ctx, user.ID, devicename)
			if err != nil {
				return nil, aerr.Wrapf(err, "get device error")
			}
		}

		podcasts, err := s.podcastsRepo.ListSubscribedPodcasts(ctx, user.ID, since)
		if err != nil {
			return nil, aerr.Wrapf(err, "list podcasts error")
		}

		return podcasts, nil
	})
}

func (s *SubscriptionsSrv) getUser(ctx context.Context, username string) (*model.User, error) {
	user, err := s.usersRepo.GetUser(ctx, username)
	if err != nil {
		return nil, aerr.Wrapf(err, "get user error")
	}

	common.TraceLazyPrintf(ctx, "getUser: user loaded")

	return user, nil
}

// getUserDevice return device from database and mark last_seen.
func (s *SubscriptionsSrv) getUserDevice(
	ctx context.Context,
	userid int64,
	devicename string,
) (*model.Device, error) {
	device, err := s.devicesRepo.GetDevice(ctx, userid, devicename)
	if err != nil {
		return device, aerr.Wrapf(err, "get device error")
	}

	common.TraceLazyPrintf(ctx, "getUserDevice: devices loaded")

	return device, nil
}

func (s *SubscriptionsSrv) createUserDevice(
	ctx context.Context,
	user *model.User,
	devicename string,
) (*model.Device, error) {
	device := model.Device{
		Name: devicename,
		User: user,
	}

	_, err := s.devicesRepo.SaveDevice(ctx, &device)
	if err != nil {
		return nil, aerr.Wrapf(err, "save device failed")
	}

	common.TraceLazyPrintf(ctx, "createUserDevice: device created")

	return s.getUserDevice(ctx, user.ID, devicename)
}

func (s *SubscriptionsSrv) getPodcasts(
	ctx context.Context,
	username, devicename string,
	since time.Time,
) (
	[]model.Podcast, error,
) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.dbi, func(ctx context.Context) ([]model.Podcast, error) {
		user, err := s.getUser(ctx, username)
		if err != nil {
			return nil, err
		}

		_, err = s.getUserDevice(ctx, user.ID, devicename)
		if err != nil {
			return nil, aerr.Wrapf(err, "get device failed")
		}

		podcasts, err := s.podcastsRepo.ListPodcasts(ctx, user.ID, since)
		if err != nil {
			return nil, aerr.Wrapf(err, "list podcasts failed")
		}

		return podcasts, nil
	})
}
