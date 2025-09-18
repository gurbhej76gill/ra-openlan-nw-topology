package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/router-architects/network-topology-service/internal/apperrors"
	store "github.com/router-architects/network-topology-service/internal/repositories"
	kafka "github.com/segmentio/kafka-go"
)

type Component interface {
	Handle(ctx context.Context, msg kafka.Message) (serviceName string, ip string, err error)
}

type ServiceDiscoveryMessage struct {
	Type        string            `json:"type"`
	ServiceName string            `json:"service_name"`
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Meta        map[string]string `json:"meta"`
}

type ServiceEventMessage struct {
	Type        string                 `json:"type"`
	ServiceName string                 `json:"service_name"`
	EventID     string                 `json:"event_id"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   string                 `json:"timestamp"`
}

type ServiceDiscoveryComponent struct {
	Store store.Store
}

func (c *ServiceDiscoveryComponent) Handle(ctx context.Context, msg kafka.Message) (string, string, error) {
	var m ServiceDiscoveryMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		return "", "", apperrors.WrapError(apperrors.CodeInvalidInput, "decode service.discovery", err)
	}
	if m.ServiceName == "" {
		return "", "", apperrors.WrapError(apperrors.CodeInvalidInput, "missing service_name", nil)
	}

	return m.ServiceName, m.IP, nil
}

type ServiceEventComponent struct {
	Store store.Store
}

func (c *ServiceEventComponent) Handle(ctx context.Context, msg kafka.Message) (string, string, error) {
	var m ServiceEventMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		return "", "", apperrors.WrapError(apperrors.CodeInvalidInput, "decode service.event", err)
	}

	if m.Timestamp != "" {
		if _, err := time.Parse(time.RFC3339, m.Timestamp); err != nil {
			// Non-fatal; return error to skip commit
			return "", "", apperrors.WrapError(apperrors.CodeInvalidInput, "invalid timestamp", err)
		}
	}

	return m.ServiceName, "", nil
}
