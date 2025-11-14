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
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type SubscriptionsSrv struct {
	db           *db.Database
	podcastsRepo repository.PodcastsRepository
	usersRepo    repository.UsersRepository
	devicesRepo  repository.DevicesRepository
}

func NewSubscriptionsSrv(i do.Injector) (*SubscriptionsSrv, error) {
	return &SubscriptionsSrv{
		db:           do.MustInvoke[*db.Database](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
	}, nil
}

// GetUserSubscriptions is simple api.
func (s *SubscriptionsSrv) GetUserSubscriptions(ctx context.Context, username string, since time.Time,
) ([]string, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	return s.getSubsctiptions(ctx, username, "", since)
}

// GetSubscriptions is simple api.
func (s *SubscriptionsSrv) GetSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}

	if devicename == "" {
		return nil, aerr.ErrValidation.WithMsg("device can't be empty")
	}

	return s.getSubsctiptions(ctx, username, devicename, since)
}

func (s *SubscriptionsSrv) ReplaceSubscriptions(ctx context.Context, //nolint:cyclop
	username, devicename string, currentSubs model.SubscribedURLs, timestamp time.Time,
) error {
	if username == "" {
		return ErrEmptyUsername
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.db, func(dbctx repository.DBContext) error {
		user, err := s.getUser(ctx, dbctx, username)
		if err != nil {
			return err
		}

		// check dev
		_, err = s.getUserDevice(ctx, dbctx, user.ID, devicename)
		if errors.Is(err, ErrUnknownDevice) {
			_, err = s.createUserDevice(ctx, dbctx, user.ID, devicename)
		}

		if err != nil {
			return err
		}

		// get all podcasts for user
		subscribed, err := s.podcastsRepo.ListPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		changes := make([]repository.PodcastDB, 0, len(currentSubs))
		// remove subscriptions found in db but not in currentSubs
		for _, sub := range subscribed {
			if sub.Subscribed && !slices.Contains(currentSubs, sub.URL) {
				sub.SetUnsubscribed(timestamp)
				changes = append(changes, sub)
			}
		}

		// add or set subscribed flag for podcast in currentSubs; update updated_at
		for _, sub := range currentSubs {
			podcast, ok := subscribed.FindPodcastByURL(sub)
			if !ok {
				podcast = repository.PodcastDB{UserID: user.ID, URL: sub, CreatedAt: timestamp}
			}

			if podcast.SetSubscribed(timestamp) {
				changes = append(changes, podcast)
			}
		}

		for _, p := range changes {
			if _, err := s.podcastsRepo.SavePodcast(ctx, dbctx, &p); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

func (s *SubscriptionsSrv) ApplySubscriptionChanges( //nolint:cyclop
	ctx context.Context,
	username, devicename string, changes *model.SubscriptionChanges, timestamp time.Time,
) error {
	if username == "" {
		return ErrEmptyUsername
	}

	if changes == nil {
		panic("changes is nil")
	}

	//nolint:wrapcheck
	return db.InTransaction(ctx, s.db, func(dbctx repository.DBContext) error {
		user, err := s.getUser(ctx, dbctx, username)
		if err != nil {
			return err
		}
		// check service
		if _, err = s.getUserDevice(ctx, dbctx, user.ID, devicename); err != nil {
			return err
		}

		subscribed, err := s.podcastsRepo.ListPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		podchanges := make([]repository.PodcastDB, 0, len(changes.Add)+len(changes.Remove))

		// removed
		for _, sub := range changes.Remove {
			if podcast, ok := subscribed.FindPodcastByURL(sub); ok {
				if podcast.SetUnsubscribed(timestamp) {
					podchanges = append(podchanges, podcast)
				}
			}
		}

		for _, sub := range changes.Add {
			podcast, ok := subscribed.FindPodcastByURL(sub)
			if !ok { // new
				podcast = repository.PodcastDB{UserID: user.ID, URL: sub, CreatedAt: timestamp}
			}

			if podcast.SetSubscribed(timestamp) {
				podchanges = append(podchanges, podcast)
			}
		}

		for _, p := range podchanges {
			if _, err := s.podcastsRepo.SavePodcast(ctx, dbctx, &p); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})
}

func (s *SubscriptionsSrv) GetSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time) (
	model.SubscriptionState, error,
) {
	if username == "" {
		return model.SubscriptionState{}, ErrEmptyUsername
	}

	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return model.SubscriptionState{}, err
	}

	state := model.SubscriptionState{
		Added:   make([]model.Podcast, 0, len(podcasts)),
		Removed: make([]model.Podcast, 0),
	}

	for _, p := range podcasts {
		podcast := model.Podcast{
			Title: p.Title,
			URL:   p.URL,
		}
		if p.Subscribed {
			state.Added = append(state.Added, podcast)
		} else {
			state.Removed = append(state.Removed, podcast)
		}
	}

	return state, nil
}

// ------------------------------------------------------

func (s *SubscriptionsSrv) getSubsctiptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.db, func(dbctx repository.DBContext) ([]string, error) {
		user, err := s.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		if devicename != "" {
			// validate is device exists when device name is given.
			_, err = s.devicesRepo.GetDevice(ctx, dbctx, user.ID, devicename)
			if errors.Is(err, repository.ErrNoData) {
				return nil, ErrUnknownDevice
			} else if err != nil {
				return nil, aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		podcasts, err := s.podcastsRepo.ListSubscribedPodcasts(ctx, dbctx, user.ID, since)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err)
		}

		return podcasts.ToURLs(), nil
	})
}

func (s *SubscriptionsSrv) getUser(ctx context.Context,
	db repository.DBContext,
	username string,
) (repository.UserDB, error) {
	user, err := s.usersRepo.GetUser(ctx, db, username)
	if errors.Is(err, repository.ErrNoData) {
		return user, ErrUnknownUser
	} else if err != nil {
		return user, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return user, nil
}

func (s *SubscriptionsSrv) getUserDevice(
	ctx context.Context,
	db repository.DBContext,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device, err := s.devicesRepo.GetDevice(ctx, db, username, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return device, ErrUnknownDevice
	} else if err != nil {
		return device, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return device, nil
}

func (s *SubscriptionsSrv) createUserDevice(
	ctx context.Context,
	dbctx repository.DBContext,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device := repository.DeviceDB{
		Name:   devicename,
		UserID: username,
	}

	_, err := s.devicesRepo.SaveDevice(ctx, dbctx, &device)
	if err != nil {
		return device, aerr.ApplyFor(ErrRepositoryError, err, "save device failed")
	}

	return s.getUserDevice(ctx, dbctx, username, devicename)
}

func (s *SubscriptionsSrv) getPodcasts(
	ctx context.Context,
	username, devicename string,
	since time.Time,
) (
	[]repository.PodcastDB, error,
) {
	//nolint:wrapcheck
	return db.InConnectionR(ctx, s.db, func(dbctx repository.DBContext) ([]repository.PodcastDB, error) {
		user, err := s.usersRepo.GetUser(ctx, dbctx, username)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownUser
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "get user failed")
		}

		_, err = s.devicesRepo.GetDevice(ctx, dbctx, user.ID, devicename)
		if errors.Is(err, repository.ErrNoData) {
			return nil, ErrUnknownDevice
		} else if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "get device failed")
		}

		podcasts, err := s.podcastsRepo.ListPodcasts(ctx, dbctx, user.ID, since)
		if err != nil {
			return nil, aerr.ApplyFor(ErrRepositoryError, err, "list podcasts failed")
		}

		return podcasts, nil
	})
}
