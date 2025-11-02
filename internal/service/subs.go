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

type Subs struct {
	db           *db.Database
	podcastsRepo repository.PodcastsRepository
	usersRepo    repository.UsersRepository
	devicesRepo  repository.DevicesRepository
}

func NewSubssServiceI(i do.Injector) (*Subs, error) {
	return &Subs{
		db:           do.MustInvoke[*db.Database](i),
		podcastsRepo: do.MustInvoke[repository.PodcastsRepository](i),
		usersRepo:    do.MustInvoke[repository.UsersRepository](i),
		devicesRepo:  do.MustInvoke[repository.DevicesRepository](i),
	}, nil
}

// GetUserSubscriptions is simple api.
func (s *Subs) GetUserSubscriptions(ctx context.Context, username string, since time.Time) ([]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := s.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	subs, err := s.podcastsRepo.ListSubscribedPodcasts(ctx, conn, user.ID, since)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return subs.ToURLs(), nil
}

// GetDeviceSubscriptions is simple api.
func (s *Subs) GetDeviceSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := s.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	_, err = s.devicesRepo.GetDevice(ctx, conn, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	podcasts, err := s.podcastsRepo.ListSubscribedPodcasts(ctx, conn, user.ID, since)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return podcasts.ToURLs(), nil
}

func (s *Subs) GetDeviceSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time,
) ([]string, []string, error) {
	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return nil, nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	var added, removed []string

	for _, p := range podcasts {
		if p.Subscribed {
			added = append(added, p.URL)
		} else {
			removed = append(removed, p.URL)
		}
	}

	return added, removed, nil
}

func (s *Subs) UpdateDeviceSubscriptions(ctx context.Context, //nolint:cyclop
	username, devicename string, subs model.SubscribedURLs, ts time.Time,
) error {
	_ = ts

	err := s.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
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
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		subscribed, err := s.podcastsRepo.ListPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		changes := make([]repository.PodcastDB, 0, len(subs))
		// removed
		for _, sub := range subscribed {
			if sub.Subscribed && !slices.Contains(subs, sub.URL) {
				sub.Subscribed = false
				changes = append(changes, sub)
			}
		}

		// added
		for _, sub := range subs {
			podcast, ok := subscribed.FindPodcastByURL(sub)
			switch {
			case ok && podcast.Subscribed:
				// ignore already subscribed podcasts
				continue
			case !ok:
				// exists but not subscribed
				podcast = repository.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
			default:
				// not subscribed
				podcast.Subscribed = true
			}

			changes = append(changes, podcast)
		}

		for _, p := range changes {
			if _, err := s.podcastsRepo.SavePodcast(ctx, dbctx, &p); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})

	return err //nolint:wrapcheck
}

func (s *Subs) UpdateDeviceSubscriptionChanges( //nolint:cyclop
	ctx context.Context,
	username, devicename string,
	changes *model.SubscriptionChanges,
) error {
	err := s.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		user, err := s.getUser(ctx, dbctx, username)
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		// check service
		if _, err = s.getUserDevice(ctx, dbctx, user.ID, devicename); err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		subscribed, err := s.podcastsRepo.ListPodcasts(ctx, dbctx, user.ID, time.Time{})
		if err != nil {
			return aerr.ApplyFor(ErrRepositoryError, err)
		}

		podchanges := make([]repository.PodcastDB, 0, len(changes.Add)+len(changes.Remove))

		// removed
		for _, sub := range changes.Remove {
			if podcast, ok := subscribed.FindPodcastByURL(sub); ok && podcast.Subscribed {
				podcast.Subscribed = false
				podchanges = append(podchanges, podcast)
			}
		}

		for _, sub := range changes.Add {
			podcast, ok := subscribed.FindPodcastByURL(sub)
			switch {
			case ok && podcast.Subscribed:
				// skip already subscribed
				continue
			case !ok:
				podcast = repository.PodcastDB{UserID: user.ID, URL: sub}
			default:
				podcast.Subscribed = true
			}

			podchanges = append(podchanges, podcast)
		}

		for _, p := range podchanges {
			if _, err := s.podcastsRepo.SavePodcast(ctx, dbctx, &p); err != nil {
				return aerr.ApplyFor(ErrRepositoryError, err)
			}
		}

		return nil
	})

	return err //nolint:wrapcheck
}

func (s *Subs) GetSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time) (
	[]model.Podcast, []string, error,
) {
	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return nil, nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	added := make([]model.Podcast, 0)
	removed := make([]string, 0)

	for _, p := range podcasts {
		if p.Subscribed {
			added = append(added, model.Podcast{
				Title: p.Title,
				URL:   p.URL,
			})
		} else {
			removed = append(removed, p.URL)
		}
	}

	return added, removed, nil
}

// ------------------------------------------------------

func (s *Subs) getUser(ctx context.Context,
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

func (s *Subs) getUserDevice(
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

func (s *Subs) createUserDevice(
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
		return device, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return s.getUserDevice(ctx, dbctx, username, devicename)
}

func (s *Subs) getPodcasts(
	ctx context.Context,
	username, devicename string,
	since time.Time,
) (
	[]repository.PodcastDB, error,
) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	defer conn.Close()

	user, err := s.usersRepo.GetUser(ctx, conn, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	_, err = s.devicesRepo.GetDevice(ctx, conn, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	podcasts, err := s.podcastsRepo.ListPodcasts(ctx, conn, user.ID, since)
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return podcasts, nil
}
