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
	"errors"
	"fmt"
	"sync"
	"time"

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/kabes/go-gpo/internal/db"
	"gitlab.com/kabes/go-gpo/internal/repository"
)

var ErrDuplicatedSID = errors.New("sid already exists")

// SessionStore represents a postgres session store implementation.
type SessionStore struct {
	db   *db.Database
	lock sync.RWMutex
	data map[any]any
	sid  string
}

// NewPostgresStore creates and returns a postgres session store.
func NewSessionStore(db *db.Database, sid string, data map[any]any) *SessionStore {
	return &SessionStore{
		db:   db,
		sid:  sid,
		data: data,
	}
}

// Set sets value to given key in session.
func (s *SessionStore) Set(key, value any) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Logger.Debug().Msgf("set session: %v=%v", key, value)

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
	log.Logger.Debug().Msgf("session release: %+v", s.data)

	// Skip encoding if the data is empty
	if len(s.data) == 0 {
		return nil
	}

	data, err := session.EncodeGob(s.data)
	if err != nil {
		return fmt.Errorf("session encode error: %w", err)
	}

	ctx := context.Background()

	err = s.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		repo := s.db.GetRepository(dbctx)

		return repo.SaveSession(ctx, s.sid, data)
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
	maxlifetime int64
	logger      zerolog.Logger
}

func NewSessionProvider(db *db.Database, maxlifetime int64) *SessionProvider {
	return &SessionProvider{
		db,
		maxlifetime,
		log.Logger.With().Str("module", "session_provider").Logger(),
	}
}

func (p *SessionProvider) Init(gclifetime int64, config string) error {
	// not in use
	_ = gclifetime
	_ = config

	return nil
}

// Read returns raw session store by session ID.
func (p *SessionProvider) Read(sid string) (session.RawStore, error) {
	ctx := context.Background()

	conn, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("get db connection error: %w", err)
	}

	repo := p.db.GetRepository(conn)

	store, err := p.readOrCreate(ctx, repo, sid)
	if err != nil {
		conn.Rollback()

		return nil, err
	}

	if err := conn.Commit(); err != nil {
		return nil, fmt.Errorf("commit changes error: %w", err)
	}

	return store, nil
}

// Exist returns true if session with given ID exists.
func (p *SessionProvider) Exist(sid string) (bool, error) {
	ctx := context.Background()

	conn, err := p.db.GetConnection(ctx)
	if err != nil {
		return false, fmt.Errorf("get db connection error: %w", err)
	}

	defer conn.Close()

	repo := p.db.GetRepository(conn)

	exists, err := repo.SessionExists(ctx, sid)
	if err != nil {
		return false, fmt.Errorf("check session %q exists error: %w", sid, err)
	}

	return exists, nil
}

// Destroy deletes a session by session ID.
func (p *SessionProvider) Destroy(sid string) error {
	ctx := context.Background()

	err := p.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		repo := p.db.GetRepository(dbctx)

		return repo.DeleteSession(ctx, sid)
	})
	if err != nil {
		return fmt.Errorf("delete session %q error: %w", sid, err)
	}

	return nil
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *SessionProvider) Regenerate(oldsid, sid string) (session.RawStore, error) {
	p.logger.Debug().Str("sid", sid).Str("old_sid", oldsid).Msg("regenerate session")

	ctx := context.Background()

	conn, err := p.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("get db connection error: %w", err)
	}

	defer conn.Rollback()

	repo := p.db.GetRepository(conn)
	if err := repo.RegenerateSession(ctx, oldsid, sid); err != nil {
		return nil, fmt.Errorf("regenerate session error: %w", err)
	}

	data, err := p.readOrCreate(ctx, repo, sid)
	if err != nil {
		return data, err
	}

	if err := conn.Commit(); err != nil {
		return nil, fmt.Errorf("commit changes error: %w", err)
	}

	return data, nil
}

// Count counts and returns number of sessions.
func (p *SessionProvider) Count() (int, error) {
	ctx := context.Background()

	conn, err := p.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("get db connection error: %w", err)
	}

	defer conn.Close()

	repo := p.db.GetRepository(conn)

	total, err := repo.CountSessions(ctx)
	if err != nil {
		return 0, fmt.Errorf("error counting records: %w", err)
	}

	return total, nil
}

// GC calls GC to clean expired sessions.
func (p *SessionProvider) GC() {
	p.logger.Debug().Msg("gc sessions")

	ctx := context.Background()

	err := p.db.InTransaction(ctx, func(dbctx repository.DBContext) error {
		repo := p.db.GetRepository(dbctx)

		return repo.CleanSessions(ctx, time.Duration(p.maxlifetime)*time.Second, 2*time.Hour) //nolint:mnd
	})
	if err != nil {
		p.logger.Error().Err(err).Msg("gc sessions error")
	}
}

func (p *SessionProvider) readOrCreate(
	ctx context.Context,
	repo repository.SessionRepository,
	sid string,
) (session.RawStore, error) {
	data, createdat, err := repo.ReadOrCreate(ctx, sid)
	if err != nil {
		return nil, fmt.Errorf("read or create session %q from db error: %w", sid, err)
	}

	var kv map[any]any

	if len(data) == 0 || createdat.Add(time.Duration(p.maxlifetime)*time.Second).Before(time.Now()) {
		kv = make(map[any]any)
	} else {
		kv, err = session.DecodeGob(data)
		if err != nil {
			return nil, fmt.Errorf("decode session error: %w", err)
		}
	}

	return NewSessionStore(p.db, sid, kv), nil
}
