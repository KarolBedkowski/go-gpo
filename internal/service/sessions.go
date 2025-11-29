package service

//
// sessions.go
// Copyright (C) 2025 Karol Będkowski <Karol Będkowski@kkomp>
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License.
//
// Based on gitea.com/go-chi/session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/aerr"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/model"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

// SessionStore represents a postgres session store implementation.
type SessionStore struct {
	db   *db.Database
	repo repository.Sessions
	lock sync.RWMutex
	data map[any]any
	sid  string
}

// Set sets value to given key in session.
func (s *SessionStore) Set(key, value any) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Logger.Debug().Msgf("set session: %v=%v", key, &value)

	s.data[key] = value

	return nil
}

// Get gets value by given key in session.
func (s *SessionStore) Get(key any) any {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.data[key]
}

// Delete delete a key from session.
func (s *SessionStore) Delete(key any) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.data, key)

	return nil
}

// ID returns current session ID.
func (s *SessionStore) ID() string {
	return s.sid
}

// save postgres session values to database.
// must call this method to save values to database.
func (s *SessionStore) Release() error {
	log.Logger.Debug().Msgf("session release: %+v", &s.data)

	ctx := log.Logger.WithContext(context.Background())

	err := db.InTransaction(ctx, s.db, func(ctx context.Context) error {
		return s.repo.SaveSession(ctx, s.sid, s.data)
	})
	if err != nil {
		return fmt.Errorf("put session into db error: %w", err)
	}

	return nil
}

// Flush deletes all session data.
func (s *SessionStore) Flush() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	clear(s.data)

	return nil
}

//-------------------------------------------------------------

// SessionProvider represents a postgres session provider implementation.
type SessionProvider struct {
	db          *db.Database
	repo        repository.Sessions
	maxlifetime time.Duration
	logger      zerolog.Logger
}

func NewSessionProvider(
	db *db.Database,
	repo repository.Sessions,
	maxlifetime time.Duration,
) *SessionProvider {
	return &SessionProvider{
		db,
		repo,
		maxlifetime,
		log.Logger,
	}
}

func (p *SessionProvider) Init(gclifetime int64, config string) error {
	// not in use
	_ = gclifetime
	_ = config

	return nil
}

// Read returns raw session store by session ID.
func (p *SessionProvider) Read(sid string) (session.RawStore, error) { //nolint:ireturn
	ctx := p.logger.WithContext(context.Background())

	storedSession, err := db.InTransactionR(ctx, p.db, func(ctx context.Context) (*model.Session, error) {
		stored, err := p.repo.ReadOrCreate(ctx, sid, p.maxlifetime)
		if err != nil {
			return nil, fmt.Errorf("read or create session %q from db error: %w", sid, err)
		}

		return stored, nil
	})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &SessionStore{
		db:   p.db,
		repo: p.repo,
		sid:  storedSession.SID,
		data: storedSession.Data,
	}, nil
}

// Exist returns true if session with given ID exists.
func (p *SessionProvider) Exist(sid string) (bool, error) {
	ctx := p.logger.WithContext(context.Background())

	exists, err := db.InConnectionR(ctx, p.db, func(ctx context.Context) (bool, error) {
		return p.repo.SessionExists(ctx, sid)
	})
	if err != nil {
		return false, fmt.Errorf("check session %q exists error: %w", sid, err)
	}

	return exists, nil
}

// Destroy deletes a session by session ID.
func (p *SessionProvider) Destroy(sid string) error {
	ctx := p.logger.WithContext(context.Background())

	err := db.InTransaction(ctx, p.db, func(ctx context.Context) error {
		return p.repo.DeleteSession(ctx, sid)
	})
	if err != nil {
		return fmt.Errorf("delete session %q error: %w", sid, err)
	}

	return nil
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *SessionProvider) Regenerate(oldsid, sid string) (session.RawStore, error) { //nolint:ireturn
	p.logger.Debug().Str("sid", sid).Str("old_sid", oldsid).Msg("regenerate session")

	ctx := p.logger.WithContext(context.Background())

	session, err := db.InTransactionR(ctx, p.db, func(ctx context.Context) (*model.Session, error) {
		if err := p.repo.RegenerateSession(ctx, oldsid, sid); err != nil {
			return nil, fmt.Errorf("regenerate session error: %w", err)
		}

		stored, err := p.repo.ReadOrCreate(ctx, sid, p.maxlifetime)
		if err != nil {
			return stored, fmt.Errorf("read or create session %q from db error: %w", sid, err)
		}

		return stored, nil
	})
	if err != nil {
		return nil, aerr.ApplyFor(ErrRepositoryError, err)
	}

	return &SessionStore{
		db:   p.db,
		repo: p.repo,
		sid:  session.SID,
		data: session.Data,
	}, nil
}

// Count counts and returns number of sessions.
func (p *SessionProvider) Count() (int, error) {
	ctx := p.logger.WithContext(context.Background())

	total, err := db.InConnectionR(ctx, p.db, func(ctx context.Context) (int, error) {
		return p.repo.CountSessions(ctx)
	})
	if err != nil {
		return 0, fmt.Errorf("error counting records: %w", err)
	}

	return total, nil
}

// GC calls GC to clean expired sessions.
func (p *SessionProvider) GC() {
	p.logger.Debug().Msg("gc sessions")

	ctx := p.logger.WithContext(context.Background())

	err := db.InTransaction(ctx, p.db, func(ctx context.Context) error {
		return p.repo.CleanSessions(ctx, p.maxlifetime, 2*time.Hour) //nolint:mnd
	})
	if err != nil {
		p.logger.Error().Err(err).Msg("gc sessions error")
	}
}
