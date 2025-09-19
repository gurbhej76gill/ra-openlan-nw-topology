package discovery

import (
	"context"
	"encoding/json"

	kgo "github.com/segmentio/kafka-go"

	"github.com/router-architects/network-topology-service/internal/apperrors"
	"github.com/router-architects/network-topology-service/internal/domain"
	kafkarouter "github.com/router-architects/network-topology-service/internal/kafka"
	"github.com/router-architects/network-topology-service/internal/logger"
	"github.com/router-architects/network-topology-service/internal/store"
)

type Component struct {
	topic string
	store *store.DiscoveryStore
}

func NewComponent(topic string, store *store.DiscoveryStore) (*Component, error) {
	if topic == "" {
		return nil, apperrors.WrapError(apperrors.CodeInternal, "discovery component: topic is required", nil)
	}
	if store == nil {
		return nil, apperrors.WrapError(apperrors.CodeInternal, "discovery component: store is nil", nil)
	}
	return &Component{topic: topic, store: store}, nil
}

func (c *Component) Topic() string { return c.topic }

func (c *Component) Handle(ctx context.Context, msg kgo.Message) error {
	_ = ctx

	var evt domain.DiscoveryEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return apperrors.WrapError(apperrors.CodeInvalidInput, "decode discovery event", err)
	}

	if evt.Type == "" {
		return apperrors.WrapError(apperrors.CodeInvalidInput, "discovery event missing service type", nil)
	}

	c.store.Upsert(evt)

	logger.GetLogger().WithFields(logger.Fields{
		"component": "service.discovery",
		"service":   evt.Type,
		"event":     evt.Event,
	}).Trace("discovery event processed")

	return nil
}

var _ kafkarouter.Component = (*Component)(nil)
