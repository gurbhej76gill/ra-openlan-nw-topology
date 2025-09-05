package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/router-architects/network-topology-service/internal/http/handlers"
	"github.com/router-architects/network-topology-service/internal/http/middlewares"
)

type ServerDeps struct {
	APIKey         string
	TopologyWindow time.Duration
	TopologyDrift  time.Duration
}

func New(app *fiber.App, deps ServerDeps, th *handlers.TopologyHandler) *fiber.App {
	// middlewares
	app.Use(middlewares.RequestLogger())
	// app.Use(middlewares.APIKeyAuth(deps.APIKey))

	// inject window/drift into context locals for handlers
	app.Use(func(c fiber.Ctx) error {
		c.Locals("topology_window", deps.TopologyWindow)
		c.Locals("topology_drift", deps.TopologyDrift)
		return c.Next()
	})

	RegisterRoutes(app, th)
	return app
}
