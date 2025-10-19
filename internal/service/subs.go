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

// GetUserSubscriptions is simple api
func (s *Subs) GetUserSubscriptions(ctx context.Context, username string, since time.Time) ([]string, error) {
	user, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	subs, err := s.repo.GetSubscribedPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return subs.ToURLs(), nil
}

// GetDeviceSubscriptions is simple api
func (s *Subs) GetDeviceSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]string, error) {
	user, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	device, err := s.repo.GetDevice(ctx, user.ID, devicename)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		return nil, ErrUnknownDevice
	}

	podcasts, err := s.repo.GetSubscribedPodcasts(ctx, user.ID, since)
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

func (s *Subs) UpdateDeviceSubscriptions(ctx context.Context,
	username, devicename string, subs []string, ts time.Time,
) error {
	logger := zerolog.Ctx(ctx)

	user, err := s.getUser(ctx, username)
	if err != nil {
		return err
	}

	device, err := s.getUserDevice(ctx, user.ID, devicename)
	if errors.Is(err, ErrUnknownDevice) {
		device, err = s.createUserDevice(ctx, user.ID, devicename)
	}

	if err != nil {
		return err
	}

	logger.Debug().Interface("device", device).Strs("subs", subs).Msg("update dev subscriptions")

	subscribed, err := s.repo.GetPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*repository.PodcastDB
	// removed
	for _, sub := range subscribed {
		if !sub.Subscribed {
			continue
		}

		if !slices.Contains(subs, sub.URL) {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")
			sub := sub.Clone()
			sub.Subscribed = false
			changes = append(changes, sub)
		}
	}

	// added
	for _, sub := range subs {
		podcast := subscribed.FindPodcastByURL(sub)
		if podcast != nil && podcast.Subscribed {
			// ignore already subscribed podcasts
			continue
		}

		if podcast == nil {
			podcast = &repository.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
		} else {
			podcast = podcast.Clone()
			podcast.Subscribed = true
		}

		changes = append(changes, podcast)

		logger.Debug().Interface("podcast", podcast).Str("sub", sub).Msg("new subscription")
	}

	if err := s.repo.SavePodcast(ctx, username, devicename, changes...); err != nil {
		return fmt.Errorf("save subscriptions error: %w", err)
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

	user, err := s.getUser(ctx, username)
	if err != nil {
		return err
	}

	device, err := s.getUserDevice(ctx, user.ID, devicename)
	if err != nil {
		return err
	}

	subscribed, err := s.repo.GetPodcasts(ctx, device.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*repository.PodcastDB

	// removed
	for _, sub := range removed {
		if podcast := subscribed.FindPodcastByURL(sub); podcast != nil && podcast.Subscribed {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")
			podcast := podcast.Clone()
			podcast.Subscribed = false
			changes = append(changes, podcast)
		}
	}

	for _, sub := range added {
		podcast := subscribed.FindPodcastByURL(sub)
		if podcast != nil && podcast.Subscribed {
			// skip already subscribed
			continue
		}

		logger.Debug().Str("podcast", sub).Msg("new subscription")

		if podcast == nil {
			podcast = &repository.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
		} else {
			podcast = podcast.Clone()
			podcast.Subscribed = true
		}

		changes = append(changes, podcast)

		logger.Debug().Interface("podcast", podcast).Str("sub", sub).Msg("new subscription")
	}

	if err := s.repo.SavePodcast(ctx, username, devicename, changes...); err != nil {
		return fmt.Errorf("save subscriptions error: %w", err)
	}

	return nil
}

func (s *Subs) GetSubsciptionChanges(ctx context.Context, username, devicename string, since time.Time) (
	[]*model.Podcast, []string, error,
) {
	podcasts, err := s.getPodcasts(ctx, username, devicename, since)
	if err != nil {
		return nil, nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	added := make([]*model.Podcast, 0)
	removed := make([]string, 0)

	for _, p := range podcasts {
		if p.Subscribed {
			added = append(added, &model.Podcast{
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

func (s *Subs) getUser(ctx context.Context, username string) (*repository.UserDB, error) {
	user, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	return user, nil
}

func (s *Subs) getUserDevice(ctx context.Context, username int64, devicename string) (*repository.DeviceDB, error) {
	device, err := s.repo.GetDevice(ctx, username, devicename)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		return nil, ErrUnknownDevice
	}

	return device, nil
}

func (s *Subs) createUserDevice(ctx context.Context, username int64, devicename string) (*repository.DeviceDB, error) {
	device := repository.DeviceDB{
		Name:   devicename,
		UserID: username,
	}

	_, err := s.repo.SaveDevice(ctx, &device)
	if err != nil {
		return nil, fmt.Errorf("save new device error: %w", err)
	}

	return s.getUserDevice(ctx, username, devicename)
}

func (s *Subs) getPodcasts(ctx context.Context, username, devicename string, since time.Time) (
	[]*repository.PodcastDB, error,
) {
	user, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	} else if user == nil {
		return nil, ErrUnknownUser
	}

	device, err := s.repo.GetDevice(ctx, user.ID, devicename)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	} else if device == nil {
		return nil, ErrUnknownDevice
	}

	podcasts, err := s.repo.GetPodcasts(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return podcasts, nil
}
