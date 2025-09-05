package middlewares

import (
	"github.com/router-architects/network-topology-service/internal/apperrors"
	"github.com/router-architects/network-topology-service/internal/logger"

	"github.com/gofiber/fiber/v3"
)

func APIKeyAuth(expected string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if expected == "" {
			// no auth configured -> deny
			return c.Status(fiber.StatusUnauthorized).JSON(errBody(apperrors.CodeUnauthorized, "missing API key"))
		}
		got := string(c.Request().Header.Peek("X-API-KEY"))
		if got == "" || got != expected {
			logger.GetLogger().WithFields(logger.Fields{
				"path": c.Path(), "method": c.Method(),
			}).Warn("unauthorized request")
			return c.Status(fiber.StatusUnauthorized).JSON(errBody(apperrors.CodeUnauthorized, "unauthorized"))
		}
		return c.Next()
	}
}

func errBody(code apperrors.ErrorCode, msg string) map[string]any {
	return map[string]any{"error": map[string]any{
		"code":    code,
		"message": msg,
	}}
}
