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

type Subs struct {
	repo *repository.Repository
}

func NewSubssService(repo *repository.Repository) *Subs {
	return &Subs{repo}
}

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

func (s *Subs) GetDeviceSubscriptions(ctx context.Context, username, devicename string, since time.Time,
) ([]*model.Subscription, error) {
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

	res := make([]*model.Subscription, 0, len(podcasts))
	for _, p := range podcasts {
		res = append(res, &model.Subscription{
			Device:    devicename,
			Podcast:   p.URL,
			UpdatedAt: p.UpdatedAt,
		})
	}

	return res, nil
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

	subscribed, err := s.repo.GetSubscribedPodcasts(ctx, user.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*model.PodcastDB
	// removed
	for _, sub := range subscribed {
		if !slices.Contains(subs, sub.URL) {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")
			sub := sub.Clone()
			sub.Subscribed = false
			changes = append(changes, sub)
		}
	}

	// added
	for _, sub := range subs {
		if subscribed.FindPodcastByURL(sub) != nil {
			// ignore already subscribed podcasts
			continue
		}

		podcast := &model.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
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
) ([][]string, error) {
	// TODO: sanitize
	logger := zerolog.Ctx(ctx)

	user, err := s.getUser(ctx, username)
	if err != nil {
		return nil, err
	}

	device, err := s.getUserDevice(ctx, user.ID, devicename)
	if err != nil {
		return nil, err
	}

	subscribed, err := s.repo.GetSubscribedPodcasts(ctx, device.ID, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*model.PodcastDB

	// removed
	for _, sub := range removed {
		if podcast := subscribed.FindPodcastByURL(sub); podcast != nil {
			logger.Debug().Interface("sub", sub).Msg("remove subscription")
			podcast := podcast.Clone()
			podcast.Subscribed = false
			changes = append(changes, podcast)
		}
	}

	for _, sub := range added {
		if subscribed.FindPodcastByURL(sub) != nil {
			// skip already subscribed
			continue
		}

		logger.Debug().Str("podcast", sub).Msg("new subscription")

		podcast := &model.PodcastDB{UserID: user.ID, URL: sub, Subscribed: true}
		changes = append(changes, podcast)

		logger.Debug().Interface("podcast", podcast).Str("sub", sub).Msg("new subscription")
	}

	if err := s.repo.SavePodcast(ctx, username, devicename, changes...); err != nil {
		return nil, fmt.Errorf("save subscriptions error: %w", err)
	}

	return nil, nil
}

// ------------------------------------------------------

func (s *Subs) getUser(ctx context.Context, username string) (*model.UserDB, error) {
	user, err := s.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	return user, nil
}

func (s *Subs) getUserDevice(ctx context.Context, username int64, devicename string) (*model.DeviceDB, error) {
	device, err := s.repo.GetDevice(ctx, username, devicename)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		return nil, ErrUnknownDevice
	}

	return device, nil
}

func (s *Subs) createUserDevice(ctx context.Context, username int64, devicename string) (*model.DeviceDB, error) {
	device := model.DeviceDB{
		Name:   devicename,
		UserID: username,
	}

	_, err := s.repo.SaveDevice(ctx, &device)
	if err != nil {
		return nil, fmt.Errorf("save new device error: %w", err)
	}

	return s.getUserDevice(ctx, username, devicename)
}
