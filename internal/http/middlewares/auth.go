package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"

	"github.com/router-architects/network-topology/internal/config"
)

func Auth(cfg config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		// If no token configured, auth pass-through (useful for local/dev)
		if cfg.AuthBearerToken == "" {
			return c.Next()
		}
		authz := string(c.Request().Header.Peek("Authorization"))
		if !strings.HasPrefix(authz, "Bearer ") {
			logrus.Warn("missing bearer token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing bearer token"})
		}
		token := strings.TrimPrefix(authz, "Bearer ")
		if token != cfg.AuthBearerToken {
			logrus.Warn("invalid bearer token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		return c.Next()
	}
}
