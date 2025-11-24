package serviceclient

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3/client"

	"github.com/router-architects/ra-openlan-nw-topology/internal/apperrors"
	"github.com/router-architects/ra-openlan-nw-topology/internal/logger"
	"github.com/router-architects/ra-openlan-nw-topology/internal/models"
	"github.com/router-architects/ra-openlan-nw-topology/internal/store"
)

type OpenAPIRequestClient interface {
	Do(ctx context.Context, method string, serviceType string, endPoint string, body io.Reader) (*client.Response, error)
	GetTimepoints(ctx context.Context, req models.TimepointRequest) ([]models.TimepointRow, error)
}

// RequestOptions describe how to reach a service instance discovered via lifecycle events.
type RequestOptions struct {
	Method      string
	ServiceType string
	Path        string
	Headers     map[string]string
	Timeout     time.Duration
}

type OpenAPIRequest struct {
	store        *store.DiscoveryStore
	client       *client.Client
	timeout      time.Duration
	internalName string
}

type OpenAPIRequestConfig struct {
	Timeout             time.Duration
	InternalServiceName string
}

const (
	defaultRequestTimeout = 3 * time.Second
)

func NewOpenApiRequest(store *store.DiscoveryStore, client *client.Client, cfg OpenAPIRequestConfig) OpenAPIRequestClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	internalName := strings.TrimSpace(cfg.InternalServiceName)
	if internalName == "" {
		internalName = "nw-topology-service"
	}

	return &OpenAPIRequest{
		store:        store,
		client:       client,
		timeout:      timeout,
		internalName: internalName,
	}
}

func (v *OpenAPIRequest) Do(ctx context.Context, method string, serviceType string, endPoint string, body io.Reader) (*client.Response, error) {
	baseLog := logger.ForFunctionality("OPENAPI-CLIENT").WithFields(logger.Fields{
		"serviceType": serviceType,
		"method":      method,
		"endpoint":    endPoint,
	})
	services := v.store.GetServices(serviceType)
	baseLog.WithField("discovered_services", len(services)).Trace("discovered services for serviceType %s", serviceType)

	for _, svc := range services {

		fullURL := strings.TrimSuffix(svc.PrivateEndPoint, "/") + endPoint
		log := baseLog.WithField("target", fullURL)

		reqCtx, cancel := context.WithTimeout(ctx, v.timeout)
		defer cancel()

		req := v.client.R().
			SetContext(reqCtx).
			SetTimeout(v.timeout).
			SetMethod(method).
			SetHeader("X-API-KEY", svc.Key).
			SetHeader("X-INTERNAL-NAME", v.internalName).
			SetURL(fullURL)

		var curlBody string
		if body != nil {
			rawBody, err := io.ReadAll(body)
			if err != nil {
				return nil, apperrors.WrapError(apperrors.CodeInternal, "failed to read body", err)
			}
			curlBody = string(rawBody)
			req = req.
				SetRawBody(rawBody).
				SetHeader("Content-Type", "application/json")
		}
		start := time.Now()
		curlCmd := buildCurlCommand(method, fullURL, v.internalName, svc.Key, curlBody)
		log.WithField("curl", curlCmd).Trace("constructed curl for service request")
		resp, err := req.Send()

		if err != nil {
			return nil, apperrors.WrapError(apperrors.CodeUnauthorized, "unauthorized", err)
		}
		log.WithFields(logger.Fields{
			"status":      resp.StatusCode(),
			"duration_ms": time.Since(start).Milliseconds(),
		}).Trace("service request completed for serviceType %s", serviceType)
		return resp, nil
	}
	baseLog.Warnf("no services discovered in store for serviceType %s", serviceType)
	return nil, apperrors.WrapError(apperrors.CodeNotFound, "Not Found", nil)
}

func buildCurlCommand(method, url, internalName, apiKey, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "curl -X %s '%s' ", method, url)
	fmt.Fprintf(&b, "-H 'X-INTERNAL-NAME: %s' ", internalName)
	fmt.Fprintf(&b, "-H 'X-API-KEY: %s' ", apiKey)
	if body != "" {
		fmt.Fprintf(&b, "-d '%s'", body)
	}
	return strings.TrimSpace(b.String())
}
