package httpserver

import (
	"github.com/gofiber/fiber/v3"
	"github.com/router-architects/network-topology/internal/http/handlers"
	"github.com/router-architects/network-topology/internal/services"
)

func RegisterRoutes(app *fiber.App, topoSvc services.TopologyService) {
	v1 := app.Group("/v1")
	handlers.NewTopologyHandler(topoSvc).Register(v1)
}
