package repository

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
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"gitea.com/go-chi/session"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

var ErrDuplicatedSID = errors.New("sid already exists")

// SessionStore represents a postgres session store implementation.
type SessionStore struct {
	db   *sqlx.DB
	lock sync.RWMutex
	data map[any]any
	sid  string
}

// NewPostgresStore creates and returns a postgres session store.
func NewSessionStore(db *sqlx.DB, sid string, data map[any]any) *SessionStore {
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
	log.Logger.Debug().Msgf("session release: %d", len(s.data))

	// Skip encoding if the data is empty
	if len(s.data) == 0 {
		return nil
	}

	data, err := session.EncodeGob(s.data)
	if err != nil {
		return fmt.Errorf("session encode error: %w", err)
	}

	_, err = s.db.Exec("UPDATE sessions SET data=?, created_at=? WHERE key=?", //nolint: noctx
		data, time.Now(), s.sid)
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

// SessionProvider represents a postgres session provider implementation.
type SessionProvider struct {
	db          *sqlx.DB
	maxlifetime int64
}

func NewSessionProvider(r *Database) *SessionProvider {
	if r == nil {
		panic("repository is nil")
	}

	if r.db == nil {
		panic("repository is not connected")
	}

	return &SessionProvider{r.db, 0}
}

// Init initializes postgres session provider.
func (p *SessionProvider) Init(maxlifetime int64, config string) error {
	_ = config
	p.maxlifetime = maxlifetime

	return nil
}

// Read returns raw session store by session ID.
func (p *SessionProvider) Read(sid string) (session.RawStore, error) {
	return p.read(context.Background(), p.db, sid)
}

// Exist returns true if session with given ID exists.
func (p *SessionProvider) Exist(sid string) (bool, error) {
	return p.exist(context.Background(), p.db, sid)
}

// Destroy deletes a session by session ID.
func (p *SessionProvider) Destroy(sid string) error {
	_, err := p.db.Exec("DELETE FROM sessions WHERE key=?", sid) //nolint: noctx
	if err != nil {
		return fmt.Errorf("delete session error: %w", err)
	}

	return nil
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *SessionProvider) Regenerate(oldsid, sid string) (session.RawStore, error) { //nolint:cyclop
	ctx := context.Background()

	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("start transaction error: %w", err)
	}

	defer tx.Rollback()

	if exist, err := p.exist(ctx, tx, sid); err == nil && exist {
		return nil, fmt.Errorf("regenerate sid %q error: %w", sid, ErrDuplicatedSID)
	} else if err != nil {
		return nil, err
	}

	exist, err := p.exist(ctx, tx, oldsid)
	if err != nil {
		return nil, err
	}

	if !exist {
		_, err := tx.ExecContext(ctx, "INSERT INTO sessions(key, data, created_at) VALUES(?, '', ?)",
			oldsid, time.Now())
		if err != nil {
			return nil, fmt.Errorf("insert new session into db error: %w", err)
		}
	} else {
		_, err := tx.ExecContext(ctx, "UPDATE sessions SET key=? WHERE key=?", sid, oldsid)
		if err != nil {
			return nil, fmt.Errorf("update session in db error; %w", err)
		}
	}

	data, err := p.read(ctx, tx, sid)
	if err != nil {
		return data, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit changes error: %w", err)
	}

	return data, nil
}

// Count counts and returns number of sessions.
func (p *SessionProvider) Count() (int, error) {
	var total int

	if err := p.db.Get(&total, "SELECT COUNT(*) AS num FROM sessions"); err != nil {
		return 0, fmt.Errorf("error counting records: %w", err)
	}

	return total, nil
}

// GC calls GC to clean expired sessions.
func (p *SessionProvider) GC() {
	_, err := p.db.Exec("DELETE FROM sessions WHERE created_at < ?", //nolint: noctx
		time.Now().Add(time.Duration(-p.maxlifetime)*time.Second))
	if err != nil {
		log.Logger.Error().Err(err).Msg("error delete old sessions")
	}

	// remove empty session older than 2 hour
	_, err = p.db.Exec("DELETE FROM sessions WHERE created_at < ? AND data is null", //nolint: noctx
		time.Now().Add(time.Duration(-2)*time.Hour))
	if err != nil {
		log.Logger.Error().Err(err).Msg("error delete old sessions")
	}
}

// Exist returns true if session with given ID exists.
func (p *SessionProvider) exist(ctx context.Context, tx queryer, sid string) (bool, error) {
	var data int

	err := tx.GetContext(ctx, &data, "SELECT 1 FROM sessions WHERE key=?", sid)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	default:
		return false, fmt.Errorf("error checking existence: %w", err)
	}
}

func (p *SessionProvider) read(ctx context.Context, tx queryer, sid string) (session.RawStore, error) {
	now := time.Now()

	var (
		data      []byte
		createdat = now
	)

	err := tx.QueryRowxContext(ctx, "SELECT data, created_at FROM sessions WHERE key=?", sid).Scan(&data, &createdat)
	if errors.Is(err, sql.ErrNoRows) {
		// create empty session
		_, err := p.db.ExecContext(ctx, "INSERT INTO sessions(key, created_at) VALUES(?, ?)", sid, now)
		if err != nil {
			return nil, fmt.Errorf("insert session into db error: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("get session data from db error: %w", err)
	}

	var kv map[any]any

	if len(data) == 0 || createdat.Add(time.Duration(p.maxlifetime)*time.Second).Before(now) {
		kv = make(map[any]any)
	} else {
		kv, err = session.DecodeGob(data)
		if err != nil {
			return nil, fmt.Errorf("decode session error: %w", err)
		}
	}

	return NewSessionStore(p.db, sid, kv), nil
}
