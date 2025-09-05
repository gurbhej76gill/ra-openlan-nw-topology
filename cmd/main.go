package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/router-architects/network-topology/adapters/postgres"
	"github.com/router-architects/network-topology/internal/config"
	"github.com/router-architects/network-topology/internal/http"
	"github.com/router-architects/network-topology/internal/logger"
	"github.com/router-architects/network-topology/internal/repositories"
	"github.com/router-architects/network-topology/internal/services"
)

func main() {
	// Root context for graceful shutdowns.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("failed to load config")
	}

	// Init logger (stdout + rotating file)
	logger.Init(cfg)

	// Postgres
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to init postgres pool")
	}
	defer pool.Close()

	// Repository
	topologyRepo := repositories.NewTopologyRepo(pool)

	// Service
	topologySvc := services.NewTopologyService(topologyRepo, cfg)

	// HTTP server
	srv := httpserver.New(cfg, topologySvc)
	go func() {
		if err := srv.Start(); err != nil {
			logrus.WithError(err).Fatal("http server failed")
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctxTimeout, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	if err := srv.Stop(ctxTimeout); err != nil {
		logrus.WithError(err).Warn("http server stop with warnings")
	}
	logrus.Info("shutdown complete")
}
