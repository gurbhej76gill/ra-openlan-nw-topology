package store

import (
	"sync"

	"github.com/router-architects/network-topology-service/internal/domain"
)

type DiscoveryStore struct {
	mu   sync.RWMutex
	data map[string]domain.DiscoveryEvent
}

func NewDiscoveryStore() *DiscoveryStore {
	return &DiscoveryStore{data: make(map[string]domain.DiscoveryEvent)}
}

func (s *DiscoveryStore) Upsert(evt domain.DiscoveryEvent) {
	if evt.Type == "" {
		return
	}

	s.mu.Lock()
	s.data[evt.Type] = evt
	s.mu.Unlock()
}

func (s *DiscoveryStore) Get(service string) (domain.DiscoveryEvent, bool) {
	s.mu.RLock()
	evt, ok := s.data[service]
	s.mu.RUnlock()
	return evt, ok
}

func (s *DiscoveryStore) Snapshot() map[string]domain.DiscoveryEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]domain.DiscoveryEvent, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}
