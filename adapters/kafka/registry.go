package kafka

import (
	"errors"
	"sort"
	"sync"
)

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

var (
	ErrEmptyTopic     = errors.New("kafka-registry: topic cannot be empty")
	ErrDuplicateTopic = errors.New("kafka-registry: topic already registered")
)

type Registry struct {
	mu    sync.RWMutex
	chans map[string]chan<- Message
}

func NewRegistry() *Registry {
	return &Registry{
		chans: make(map[string]chan<- Message),
	}
}

func (r *Registry) Register(topic string, ch chan<- Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.chans[topic]; exists {
		return ErrDuplicateTopic
	}

	r.chans[topic] = ch
	return nil
}

func (r *Registry) Channel(topic string) (chan<- Message, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ch, exists := r.chans[topic]
	return ch, exists
}

func (r *Registry) Topics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.chans))
	for t := range r.chans {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
