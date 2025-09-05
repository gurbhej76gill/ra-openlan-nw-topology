package httpserver

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"

	"github.com/router-architects/network-topology/internal/config"
	"github.com/router-architects/network-topology/internal/http/handlers"
	"github.com/router-architects/network-topology/internal/http/middlewares"
	"github.com/router-architects/network-topology/internal/services"
)

type server struct {
	app *fiber.App
	cfg config.Config
}

func New(cfg config.Config, topoSvc services.TopologyService) *server {
	app := fiber.New(fiber.Config{
		AppName:            cfg.AppName,
		EnableIPValidation: true,
	})
	// Middlewares (order matters)
	app.Use(middlewares.RequestLogger())
	app.Use(middlewares.Auth(cfg))

	// Routes
	handlers.Register(app, topoSvc)

	return &server{app: app, cfg: cfg}
}

func (s *server) Start() error {
	logrus.WithField("addr", s.cfg.HTTPAddr).Info("server starting")
	return s.app.Listen(s.cfg.HTTPAddr)
}

func (s *server) Stop(ctx context.Context) error {
	logrus.Info("server stopping")
	return s.app.ShutdownWithContext(ctx)
}
