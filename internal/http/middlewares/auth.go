package middlewares

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/router-architects/network-topology-service/internal/apperrors"
	"github.com/router-architects/network-topology-service/internal/logger"
	"github.com/router-architects/network-topology-service/internal/security"
)

func APIKeyAuth(expected string, validator security.TokenValidator) fiber.Handler {
	return func(c fiber.Ctx) error {
		log := logger.ForFunctionality("MIDDLEWARE")
		if expected == "" {
			// no auth configured -> deny
			return writeAuthError(c, apperrors.CodeUnauthorized)
		}
		got := string(c.Request().Header.Peek("X-API-KEY"))
		internalHeader := c.Get("X-INTERNAL-NAME")
		if internalHeader != "" {
			if got == "" || got != expected {
				return writeAuthError(c, apperrors.CodeUnauthorized)
			}
		} else {
			if validator == nil {
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

			if err := validator.Validate(context.Background(), subToken); err != nil {
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
	}
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
