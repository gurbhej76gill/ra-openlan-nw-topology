package middlewares

import (
	"time"

	"github.com/router-architects/network-topology-service/internal/logger"

	"github.com/gofiber/fiber/v3"
)

func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		lat := time.Since(start)

		logger.GetLogger().WithFields(logger.Fields{
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"latency_ms": lat.Milliseconds(),
		}).Info("http_request")

		return err
	}
}
