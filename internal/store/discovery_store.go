package store

import (
	"strings"
	"sync"

	"github.com/router-architects/ra-openlan-nw-topology/internal/domain"
)

type DiscoveryStore struct {
	mu                sync.RWMutex
	byPrivateEndpoint map[string]domain.DiscoveryEvent
}

func NewDiscoveryStore() *DiscoveryStore {
	return &DiscoveryStore{byPrivateEndpoint: make(map[string]domain.DiscoveryEvent)}
}

func (s *DiscoveryStore) Upsert(evt domain.DiscoveryEvent) {
	privateEndpoint := strings.TrimSpace(evt.PrivateEndPoint)
	svcType := strings.TrimSpace(evt.Type)
	if privateEndpoint == "" || svcType == "" {
		return
	}

	eventName := strings.ToLower(strings.TrimSpace(evt.Event))

	s.mu.Lock()
	switch eventName {
	case "keep-alive", "join":
		s.byPrivateEndpoint[privateEndpoint] = evt
	default:
		delete(s.byPrivateEndpoint, privateEndpoint)
	}
	s.mu.Unlock()
}

func (s *DiscoveryStore) Get(serviceType string) []domain.DiscoveryEvent {
	serviceType = strings.TrimSpace(serviceType)
	if serviceType == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.DiscoveryEvent, 0, len(s.byPrivateEndpoint))
	for _, evt := range s.byPrivateEndpoint {
		if strings.EqualFold(evt.Type, serviceType) {
			out = append(out, evt)
		}
	}
	return out
}

func (s *DiscoveryStore) Snapshot() map[string]domain.DiscoveryEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]domain.DiscoveryEvent, len(s.byPrivateEndpoint))
	for k, v := range s.byPrivateEndpoint {
		out[k] = v
	}
	return out
}

func (s *DiscoveryStore) GetServices(serviceType string) []domain.DiscoveryEvent {
	serviceType = strings.TrimSpace(serviceType)
	if serviceType == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.DiscoveryEvent, 0, len(s.byPrivateEndpoint))
	for _, evt := range s.byPrivateEndpoint {
		if strings.EqualFold(evt.Type, serviceType) {
			out = append(out, evt)
		}
	}
	return out
}
