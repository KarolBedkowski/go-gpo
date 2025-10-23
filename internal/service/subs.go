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

	"github.com/rs/zerolog"
	"gitlab.com/kabes/go-gpodder/internal/model"
	"gitlab.com/kabes/go-gpodder/internal/repository"
)

var ErrUnknownPodcast = errors.New("unknown podcast")

type Subs struct {
	repo *repository.Repository
}

func NewSubssService(repo *repository.Repository) *Subs {
	return &Subs{repo}
}

// GetUserSubscriptions is simple api.
func (s *Subs) GetUserSubscriptions(ctx context.Context, username string, since time.Time) ([]string, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	subs, err := tx.GetSubscribedPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return subs.ToURLs(), nil
}

// GetDeviceSubscriptions is simple api.
func (s *Subs) GetDeviceSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownUser
	} else if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}

	_, err = tx.GetDevice(ctx, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	podcasts, err := tx.GetSubscribedPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return podcasts.ToURLs(), nil
}

func (s *Subs) GetDeviceSubscriptionChanges(ctx context.Context, username, devicename string, since time.Time,
) ([]string, []string, error) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	podcasts, err := s.getPodcasts(ctx, tx, username, devicename, since)
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

func (s *Subs) UpdateDeviceSubscriptions(ctx context.Context,
	username, devicename string, subs []string, ts time.Time,
) error {
	_ = ts
	logger := zerolog.Ctx(ctx)

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := s.getUser(ctx, tx, username)
	if err != nil {
		return err
	}

	device, err := s.getUserDevice(ctx, tx, user.ID, devicename)
	if errors.Is(err, ErrUnknownDevice) {
		device, err = s.createUserDevice(ctx, tx, user.ID, devicename)
	}

	if err != nil {
		return err
	}

	logger.Debug().Interface("device", device).Strs("subs", subs).Msg("update dev subscriptions")

	subscribed, err := tx.GetPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	changes := make([]repository.PodcastDB, 0, len(subs))
	// removed
	for _, sub := range subscribed {
		if !sub.Subscribed {
			continue
		}

		if !slices.Contains(subs, sub.URL) {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")

			sub.Subscribed = false
			changes = append(changes, sub)
		}
	}

	// added
	for _, sub := range subs {
		podcast, ok := subscribed.FindPodcastByURL(sub)
		if ok && podcast.Subscribed {
			// ignore already subscribed podcasts
			continue
		}

		if !ok {
			podcast = repository.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
		}

		podcast.Subscribed = true
		changes = append(changes, podcast)

		logger.Debug().Interface("podcast", podcast).Str("sub", sub).Msg("new subscription")
	}

	if err := tx.SavePodcast(ctx, username, devicename, changes...); err != nil {
		return fmt.Errorf("save subscriptions error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

func (s *Subs) UpdateDeviceSubscriptionChanges(
	ctx context.Context,
	username, devicename string,
	added, removed []string,
) error {
	// TODO: sanitize
	logger := zerolog.Ctx(ctx)

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	user, err := s.getUser(ctx, tx, username)
	if err != nil {
		return err
	}

	device, err := s.getUserDevice(ctx, tx, user.ID, devicename)
	if err != nil {
		return err
	}

	subscribed, err := tx.GetPodcasts(ctx, device.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	changes := make([]repository.PodcastDB, 0, len(added)+len(removed))

	// removed
	for _, sub := range removed {
		if podcast, ok := subscribed.FindPodcastByURL(sub); ok && podcast.Subscribed {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")

			podcast.Subscribed = false
			changes = append(changes, podcast)
		}
	}

	for _, sub := range added {
		podcast, ok := subscribed.FindPodcastByURL(sub)
		if ok && podcast.Subscribed {
			// skip already subscribed
			continue
		}

		logger.Debug().Str("podcast", sub).Msg("new subscription")

		if !ok {
			podcast = repository.PodcastDB{UserID: user.ID, URL: sub}
		}

		podcast.Subscribed = true

		changes = append(changes, podcast)

		logger.Debug().Interface("podcast", podcast).Str("sub", sub).Msg("new subscription")
	}

	if err := tx.SavePodcast(ctx, username, devicename, changes...); err != nil {
		return fmt.Errorf("save subscriptions error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

func (s *Subs) GetSubsciptionChanges(ctx context.Context, username, devicename string, since time.Time) (
	[]model.Podcast, []string, error,
) {
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("start tx error: %w", err)
	}

	defer tx.Close()

	podcasts, err := s.getPodcasts(ctx, tx, username, devicename, since)
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

func (s *Subs) getUser(ctx context.Context, tx repository.Transaction, username string) (repository.UserDB, error) {
	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return user, ErrUnknownUser
	} else if err != nil {
		return user, fmt.Errorf("get user error: %w", err)
	}

	return user, nil
}

func (s *Subs) getUserDevice(
	ctx context.Context,
	tx repository.Transaction,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device, err := tx.GetDevice(ctx, username, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return device, ErrUnknownDevice
	} else if err != nil {
		return device, fmt.Errorf("get device error: %w", err)
	}

	return device, nil
}

func (s *Subs) createUserDevice(
	ctx context.Context,
	tx repository.Transaction,
	username int64,
	devicename string,
) (repository.DeviceDB, error) {
	device := repository.DeviceDB{
		Name:   devicename,
		UserID: username,
	}

	_, err := tx.SaveDevice(ctx, &device)
	if err != nil {
		return device, fmt.Errorf("save new device error: %w", err)
	}

	return s.getUserDevice(ctx, tx, username, devicename)
}

func (s *Subs) getPodcasts(
	ctx context.Context,
	tx repository.Transaction,
	username, devicename string,
	since time.Time,
) (
	[]repository.PodcastDB, error,
) {
	user, err := tx.GetUser(ctx, username)
	if errors.Is(err, repository.ErrNoData) {
		return nil, fmt.Errorf("get user error: %w", err)
	} else if err != nil {
		return nil, ErrUnknownUser
	}

	_, err = tx.GetDevice(ctx, user.ID, devicename)
	if errors.Is(err, repository.ErrNoData) {
		return nil, ErrUnknownDevice
	} else if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	podcasts, err := tx.GetPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return podcasts, nil
}
