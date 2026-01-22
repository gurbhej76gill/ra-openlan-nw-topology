package middlewares

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/router-architects/ra-openlan-nw-topology/adapters/apperrors"
	"github.com/router-architects/ra-openlan-nw-topology/adapters/logger"
)

type TokenValidator interface {
	Validate(ctx context.Context, token string) error
}

type TopologyAuthMiddleware struct {
	APIKey         string
	TokenValidator TokenValidator
}

func NewTopologyAuthMiddleware(apiKey string, validator TokenValidator) *TopologyAuthMiddleware {
	return &TopologyAuthMiddleware{
		APIKey:         apiKey,
		TokenValidator: validator,
	}
}

func (t *TopologyAuthMiddleware) TopologyAuth(c fiber.Ctx) error {
	log := logger.GetLoggerThreadId("SERVER")
	if t.APIKey == "" {
		fmt.Printf("apiKey name : %s\n", t.APIKey)
		return writeAuthError(c, apperrors.CodeUnauthorized)
	}
	got := string(c.Request().Header.Peek("X-API-KEY"))
	internalHeader := c.Get("X-INTERNAL-NAME")
	if internalHeader != "" {
		if got == "" || got != t.APIKey {
			return writeAuthError(c, apperrors.CodeUnauthorized)
		}
	} else {
		if t.TokenValidator == nil {
			return writeAuthError(c, apperrors.CodeUnauthorized)
		}

		authHeader := c.Get("Authorization", "")
		if authHeader == "" {
			return writeAuthError(c, apperrors.CodeUnauthorized)
		}

		subToken := strings.TrimSpace(authHeader)
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(subToken, bearerPrefix) {
			subToken = strings.TrimSpace(subToken[len(bearerPrefix):])
		}

		if err := t.TokenValidator.Validate(c.Context(), subToken); err != nil {
			if log != nil {
				log.WithFields(logger.Fields{
					"path":   c.Path(),
					"method": c.Method(),
				}).WithError(err).Warn("subscription token validation failed")
			}

			if appErr, ok := err.(*apperrors.Error); ok {
				return writeAuthError(c, appErr.Code)
			}

			return writeAuthError(c, apperrors.CodeUnauthorized)
		}
	}

	return c.Next()
	// }
}

func writeAuthError(c fiber.Ctx, code apperrors.ErrorCode) error {
	info := apperrors.GetHTTPErrorInfo(code)
	body := map[string]any{
		"ErrorCode":        info.Status,
		"ErrorDescription": fmt.Sprintf("%d: %s", info.Status, info.Description),
		"ErrorDetails":     c.Method(),
	}
	return c.Status(info.Status).JSON(body)
}
