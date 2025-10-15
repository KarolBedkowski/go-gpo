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

func (s *Subs) GetUserSubscriptions(ctx context.Context, userID string, since time.Time) ([]string, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	subs, err := s.repo.GetUserSubscriptions(ctx, user.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return subs, nil
}

func (s *Subs) GetDeviceSubscriptions(ctx context.Context, userID, deviceID string, since time.Time,
) ([]model.Subscription, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	device, err := s.repo.GetDevice(ctx, user.ID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		return nil, ErrUnknownDevice
	}

	subs, err := s.repo.GetSubscriptions(ctx, device.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	return subs, nil
}

func (s *Subs) UpdateDeviceSubscriptions(ctx context.Context,
	userID, deviceID string, subs []string, ts time.Time,
) error {
	logger := zerolog.Ctx(ctx)

	user, err := s.getUser(ctx, userID)
	if err != nil {
		return err
	}

	device, err := s.getUserDevice(ctx, user.ID, deviceID)
	if errors.Is(err, ErrUnknownDevice) {
		device, err = s.createUserDevice(ctx, user.ID, deviceID)
	}

	if err != nil {
		return err
	}

	usubs, err := s.repo.GetSubscriptions(ctx, device.ID, time.Time{})
	if err != nil {
		return fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*model.Subscription
	for _, s := range usubs {
		if !slices.Contains(subs, s.Podcast) {
			logger.Debug().Interface("sub", s).Msg("remove subscription")
			changes = append(changes, s.NewAction(model.ActionUnsubscribe))
		}
	}

	for _, s := range subs {
		if !hasSubscriptions(usubs, s) {
			logger.Debug().Str("podcast", s).Msg("new subscription")
			changes = append(changes, model.NewSubscription(device.ID, s, model.ActionSubscribe))
		}
	}

	if err := s.repo.SaveSubscription(ctx, changes...); err != nil {
		return fmt.Errorf("save subscriptions error: %w", err)
	}

	return nil
}

func (s *Subs) UpdateDeviceSubscriptionChanges(ctx context.Context, userID, deviceID string, added, removed []string) ([][]string, error) {
	// TODO: sanitize
	logger := zerolog.Ctx(ctx)

	user, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	device, err := s.getUserDevice(ctx, user.ID, deviceID)
	if err != nil {
		return nil, err
	}

	usubs, err := s.repo.GetSubscriptions(ctx, device.ID, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("get subscriptions error: %w", err)
	}

	var changes []*model.Subscription
	for _, s := range usubs {
		if slices.Contains(removed, s.Podcast) {
			logger.Debug().Interface("sub", s).Msg("remove subscription")
			changes = append(changes, s.NewAction(model.ActionUnsubscribe))
		}
	}

	for _, s := range added {
		if !hasSubscriptions(usubs, s) {
			logger.Debug().Str("podcast", s).Msg("new subscription")
			changes = append(changes, model.NewSubscription(device.ID, s, model.ActionSubscribe))
		}
	}

	if err := s.repo.SaveSubscription(ctx, changes...); err != nil {
		return nil, fmt.Errorf("save subscriptions error: %w", err)
	}

	return nil, nil
}

func (s *Subs) getUser(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user error: %w", err)
	}
	if user == nil {
		return nil, ErrUnknownUser
	}

	return user, err
}

func (s *Subs) getUserDevice(ctx context.Context, userID int, deviceID string) (*model.Device, error) {
	device, err := s.repo.GetDevice(ctx, userID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device error: %w", err)
	}

	if device == nil {
		return nil, ErrUnknownDevice
	}

	return device, nil
}

func (s *Subs) createUserDevice(ctx context.Context, userID int, deviceID string) (*model.Device, error) {
	device := model.Device{
		Name:   deviceID,
		UserID: userID,
	}

	_, err := s.repo.SaveDevice(ctx, &device)
	if err != nil {
		return nil, fmt.Errorf("save new device error: %w", err)
	}

	return s.getUserDevice(ctx, userID, deviceID)
}

func hasSubscriptions(subs []model.Subscription, podcast string) bool {
	for _, s := range subs {
		if s.Podcast == podcast {
			return true
		}
	}

	return false
}
