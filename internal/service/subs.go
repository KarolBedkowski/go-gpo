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
	"fmt"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

type Subs struct {
	db *db.Database
}

func NewSubssService(db *db.Database) *Subs {
	return &Subs{db}
}

func NewSubssServiceI(i do.Injector) (*Subs, error) {
	db := do.MustInvoke[*db.Database](i)

	return &Subs{db}, nil
}

// GetUserSubscriptions is simple api.
func (s *Subs) GetUserSubscriptions(ctx context.Context, username string, since time.Time) ([]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := s.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	subs, err := repo.ListSubscribedPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return subs.ToURLs(), nil
}

// GetDeviceSubscriptions is simple api.
func (s *Subs) GetDeviceSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	conn, err := s.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := s.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	_, err = repo.GetDevice(ctx, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	podcasts, err := repo.ListSubscribedPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return podcasts.ToURLs(), nil
}

func (s *Subs) GetDeviceSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time,
) ([]string, []string, error) {
	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return nil, nil, fmt.Errorf("get subscriptions error: %w", err)
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

	err := s.db.InTransaction(ctx, func(db repository.DBContext) error {
		repo := s.db.GetRepository(db)

		user, err := s.getUser(ctx, repo, username)
		if err != nil {
			return err
		}

		// check dev
		_, err = s.getUserDevice(ctx, repo, user.ID, devicename)
		if errors.Is(err, ErrUnknownDevice) {
			_, err = s.createUserDevice(ctx, repo, user.ID, devicename)
		}

		if err != nil {
			return err
		}

		subscribed, err := repo.ListPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return fmt.Errorf("get subscriptions error: %w", err)
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

		if err := repo.SavePodcast(ctx, username, devicename, changes...); err != nil {
			return fmt.Errorf("save subscriptions error: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update subs error: %w", err)
	}

	return nil
}

func (s *Subs) UpdateDeviceSubscriptionChanges( //nolint:cyclop
	ctx context.Context,
	username, devicename string,
	changes *model.SubscriptionChanges,
) error {
	err := s.db.InTransaction(ctx, func(db repository.DBContext) error {
		repo := s.db.GetRepository(db)

		user, err := s.getUser(ctx, repo, username)
		if err != nil {
			return err
		}

		// check service
		if _, err = s.getUserDevice(ctx, repo, user.ID, devicename); err != nil {
			return err
		}

		subscribed, err := repo.ListPodcasts(ctx, user.ID, time.Time{})
		if err != nil {
			return fmt.Errorf("get subscriptions error: %w", err)
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

		if err := repo.SavePodcast(ctx, username, devicename, podchanges...); err != nil {
			return fmt.Errorf("save subscriptions error: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update subs error: %w", err)
	}

	return nil
}

func (s *Subs) GetSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time) (
	[]model.Podcast, []string, error,
) {
	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return nil, nil, fmt.Errorf("get subscriptions error: %w", err)
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

func (s *Subs) getUser(ctx context.Context, repo repository.Repository, username string) (repository.UserDB, error) {
	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return user, ErrUnknownUser
	} else if err != nil {
		return user, fmt.Errorf("get user error: %w", err)
	}

	return user, nil
}

func (s *Subs) getUserDevice(
	ctx context.Context,
	repo repository.Repository,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device, err := repo.GetDevice(ctx, username, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return device, ErrUnknownDevice
	} else if err != nil {
		return device, fmt.Errorf("get device error: %w", err)
	}

	return device, nil
}

func (s *Subs) createUserDevice(
	ctx context.Context,
	repo repository.Repository,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device := repository.DeviceDB{
		Name:   devicename,
		UserID: username,
	}

	_, err := repo.SaveDevice(ctx, &device)
	if err != nil {
		return device, fmt.Errorf("save new device error: %w", err)
	}

	return s.getUserDevice(ctx, repo, username, devicename)
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
		return nil, fmt.Errorf("get connection error: %w", err)
	}

	defer conn.Close()

	repo := s.db.GetRepository(conn)

	user, err := repo.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, fmt.Errorf("get user error: %w", err)
	} else if err != nil {
		return nil, ErrUnknownUser
	}

	_, err = repo.GetDevice(ctx, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	podcasts, err := repo.ListPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return podcasts, nil
}
