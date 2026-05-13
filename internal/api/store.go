package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bootlegger/internal/core"
	"bootlegger/internal/source"
)

// SessionRecord holds the live state of a managed session
type SessionRecord struct {
	Session   *core.Session
	State     core.SessionState
	Metadata  *source.VideoMetadata
	Tracks    []core.Track
	Events    chan core.Event // pipeline events; nil when no pipeline running
	cancel    context.CancelFunc
	CreatedAt time.Time
	mu        sync.RWMutex
}

// SessionStore is a thread-safe in-memory store of active sessions
type SessionStore struct {
	mu      sync.RWMutex
	records map[string]*SessionRecord
}

func newSessionStore() *SessionStore {
	return &SessionStore{records: make(map[string]*SessionRecord)}
}

func (ss *SessionStore) create(baseDir string) *SessionRecord {
	session := core.NewSession(baseDir)
	rec := &SessionRecord{
		Session:   session,
		State:     core.StateReady,
		CreatedAt: time.Now(),
	}
	ss.mu.Lock()
	ss.records[session.ID] = rec
	ss.mu.Unlock()
	return rec
}

func (ss *SessionStore) get(id string) (*SessionRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	rec, ok := ss.records[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return rec, nil
}

func (ss *SessionStore) delete(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if rec, ok := ss.records[id]; ok {
		if rec.cancel != nil {
			rec.cancel()
		}
	}
	delete(ss.records, id)
}
