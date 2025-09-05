package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		logrus.WithFields(logrus.Fields{
			"method": method,
			"path":   path,
			"ip":     ip,
		}).Info("incoming request")

		if err := c.Next(); err != nil {
			// Fiber will handle returned error; we log here as well.
			logrus.WithError(err).Error("handler error")
			return err
		}

		lat := time.Since(start)
		logrus.WithFields(logrus.Fields{
			"method":  method,
			"path":    path,
			"status":  c.Response().StatusCode(),
			"latency": lat.String(),
		}).Debug("request completed")
		return nil
	}
}
