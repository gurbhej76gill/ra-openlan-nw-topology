package security

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/router-architects/network-topology-service/internal/adapters/serviceclient"
	"github.com/router-architects/network-topology-service/internal/apperrors"
	"github.com/router-architects/network-topology-service/internal/logger"
)

// TokenValidator validates subscription tokens against an upstream security service.
type TokenValidator interface {
	Validate(ctx context.Context, token string) error
}

// ValidatorConfig tunes runtime behavior of the OWSEC token validator.
type ValidatorConfig struct {
	Timeout             time.Duration
	InternalServiceName string
}

type owsecValidator struct {
	client serviceclient.OpenAPIRequestClient
}

const (
	defaultTimeout = 3 * time.Second
	owsecService   = "owsec"
)

// NewTokenValidator constructs a TokenValidator that calls the owsec /validateSubToken API.
func NewTokenValidator(client serviceclient.OpenAPIRequestClient) TokenValidator {

	return &owsecValidator{
		client: client,
	}
}

func (v *owsecValidator) Validate(ctx context.Context, rawToken string) error {
	log := logger.ForFunctionality("VALIDATOR")
	token := strings.TrimSpace(rawToken)
	if token == "" {
		info := apperrors.GetHTTPErrorInfo(apperrors.CodeUnauthorized)
		return apperrors.WrapError(apperrors.CodeUnauthorized, info.Description, nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	validateSubTokenURL := "/api/v1/validateSubToken?token=" + url.QueryEscape(token)

	resp, err := v.client.Do(ctx, fiber.MethodGet, owsecService, validateSubTokenURL, nil)

	if resp != nil {
		defer resp.Close()
	}

	if err != nil || resp == nil || resp.StatusCode() != fiber.StatusOK {
		validateTokenURL := "/api/v1/validateToken?token=" + url.QueryEscape(token)

		fallbackResp, err := v.client.Do(ctx, fiber.MethodGet, owsecService, validateTokenURL, nil)
		if fallbackResp != nil {
			defer fallbackResp.Close()
		}
		if err != nil {
			if log != nil {
				log.WithFields(logger.Fields{
					"service":   owsecService,
					"url":       validateTokenURL,
					"operation": "validateToken",
				}).WithError(err).Error("validateToken request failed")
			}
			info := apperrors.GetHTTPErrorInfo(apperrors.CodeUnauthorized)
			return apperrors.WrapError(apperrors.CodeUnauthorized, info.Description, err)
		}
		if fallbackResp == nil || fallbackResp.StatusCode() != fiber.StatusOK {
			info := apperrors.GetHTTPErrorInfo(apperrors.CodeUnauthorized)
			return apperrors.WrapError(apperrors.CodeUnauthorized, info.Description, nil)
		}
	}

	return nil
}
