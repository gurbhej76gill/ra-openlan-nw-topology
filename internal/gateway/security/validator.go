package security

import (
	"context"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/router-architects/ra-openlan-nw-topology/adapters/apperrors"
	"github.com/router-architects/ra-openlan-nw-topology/adapters/httpclient"
	"github.com/router-architects/ra-openlan-nw-topology/adapters/logger"
	"github.com/router-architects/ra-openlan-nw-topology/internal/store"
)

// TokenValidator validates subscription tokens against an upstream security service.
type TokenValidator interface {
	Validate(ctx context.Context, token string) error
}

type owsecValidator struct {
	store  *store.DiscoveryStore
	client httpclient.OpenAPIRequestClient
}

const (
	owsecService = "owsec"
)

func NewTokenValidator(client httpclient.OpenAPIRequestClient, store *store.DiscoveryStore) TokenValidator {

	return &owsecValidator{
		store:  store,
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

	services := v.store.GetServices(owsecService)

	validateSubTokenURL := "/api/v1/validateSubToken?token=" + url.QueryEscape(token)

	resp, err := v.client.Do(ctx, fiber.MethodGet, owsecService, validateSubTokenURL, nil, services)

	if resp != nil {
		defer resp.Close()
	}

	if err != nil || resp == nil || resp.StatusCode() != fiber.StatusOK {
		validateTokenURL := "/api/v1/validateToken?token=" + url.QueryEscape(token)

		fallbackResp, err := v.client.Do(ctx, fiber.MethodGet, owsecService, validateTokenURL, nil, services)
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
