package kafka

import (
	"context"
	"errors"
	"sort"
	"sync"

	kgo "github.com/segmentio/kafka-go"
)

// Component processes Kafka messages for a single topic.
type Component interface {
	Topic() string
	Handle(ctx context.Context, msg kgo.Message) error
}

var (
	ErrNilComponent   = errors.New("kafka: component is nil")
	ErrEmptyTopic     = errors.New("kafka: component topic is empty")
	ErrDuplicateTopic = errors.New("kafka: component already registered for topic")
)

type Registry struct {
	mu         sync.RWMutex
	components map[string]Component
}

func NewRegistry() *Registry {
	return &Registry{components: make(map[string]Component)}
}

func (r *Registry) Register(c Component) error {
	if c == nil {
		return ErrNilComponent
	}

	topic := c.Topic()
	if topic == "" {
		return ErrEmptyTopic
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[topic]; exists {
		return ErrDuplicateTopic
	}
	r.components[topic] = c
	return nil
}

func (r *Registry) GetComponent(topic string) (Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.components[topic]
	return c, ok
}

func (r *Registry) Topics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topics := make([]string, 0, len(r.components))
	for topic := range r.components {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics
}
