package repositories

import (
	"context"
	"errors"
	"sync"
)

// Store persists serviceName -> ip along with optional payload.
// Implementations must be concurrency-safe.
type Store interface {
	Save(ctx context.Context, serviceName string, ip string, payload []byte) error
}

// InMemoryStore is a thread-safe in-memory impl for tests/dev.
type InMemoryStore struct {
	mu   sync.RWMutex
	ipBy map[string]string
	// optionally last payload (debugging/inspection)
	payloadBy map[string][]byte
}

// NewInMemory returns a new empty in-memory store.
func NewInMemory() *InMemoryStore {
	return &InMemoryStore{
		ipBy:      make(map[string]string),
		payloadBy: make(map[string][]byte),
	}
}

func (s *InMemoryStore) Save(_ context.Context, serviceName, ip string, payload []byte) error {
	if serviceName == "" {
		return errors.New("serviceName required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ip != "" {
		s.ipBy[serviceName] = ip
	}
	if payload != nil {
		s.payloadBy[serviceName] = append([]byte(nil), payload...)
	}
	return nil
}

// GetIP returns the latest stored IP for a service.
func (s *InMemoryStore) GetIP(serviceName string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip, ok := s.ipBy[serviceName]
	return ip, ok
}

// ListServices returns a shallow copy of service->ip map.
func (s *InMemoryStore) ListServices() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]string, len(s.ipBy))
	for k, v := range s.ipBy {
		cp[k] = v
	}
	return cp
}
