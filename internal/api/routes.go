package api

import (
	"github.com/router-architects/ra-openlan-nw-topology/internal/api/handlers"
	"github.com/router-architects/ra-openlan-nw-topology/internal/api/middlewares"

	"github.com/gofiber/fiber/v3"
)

func (s *ServerDeps) RegisterRoutes(app *fiber.App, th *handlers.TopologyHandler) {
	v1 := app.Group("/v1", middlewares.APIKeyAuth(s.APIKey, s.TokenValidator))
	v1.Get("/topology", th.GetTopology)
}
