package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	kafkaproducer "github.com/router-architects/network-topology-service/adapters/kafka"
	"github.com/router-architects/network-topology-service/adapters/postgres"
	"github.com/router-architects/network-topology-service/internal/config"
	"github.com/router-architects/network-topology-service/internal/http"
	"github.com/router-architects/network-topology-service/internal/http/handlers"
	"github.com/router-architects/network-topology-service/internal/kafka"
	"github.com/router-architects/network-topology-service/internal/logger"
	"github.com/router-architects/network-topology-service/internal/repositories"
	"github.com/router-architects/network-topology-service/internal/services"
	discoverycomponent "github.com/router-architects/network-topology-service/internal/services/discovery"
	"github.com/router-architects/network-topology-service/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// logrus + lumberjack
	ll := &lumberjack.Logger{
		Filename:   cfg.LogPath,
		MaxSize:    cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAgeDays,
		Compress:   true,
	}
	log := logrus.New()
	log.SetOutput(ll)
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	entry := logrus.NewEntry(log)
	logger.SetLogger(logger.LogrusAdapter{entry})
	logger.SetLogger(logger.GetLogger().WithField("app", cfg.AppName))

	// pgx pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{
		Host:            cfg.PGHost,
		Port:            cfg.PGPort,
		User:            cfg.PGUser,
		Password:        cfg.PGPassword,
		Database:        cfg.PGDatabase,
		SSLMode:         cfg.PGSSLMode,
		MaxConns:        cfg.PGMaxConns,
		MinConns:        cfg.PGMinConns,
		MaxConnLifetime: cfg.PGMaxLifetime,
	})
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to connect postgres")
	}

	discoveryStore := store.NewDiscoveryStore()

	kProducer, err := kafkaproducer.NewProducerForTopic(cfg, cfg.KafkaTopicCmd)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to init kafka producer")
	}

	lcProducer, err := kafkaproducer.NewProducerForTopic(cfg, cfg.KafkaTopicLifecycle) // for lifecycle events
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to init kafka producer (lifecycle)")
	}

	registry := kafka.NewRegistry()
	discoveryComponent, err := discoverycomponent.NewComponent(cfg.KafkaTopicLifecycle, discoveryStore)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to create discovery component")
	}
	if err := registry.Register(discoveryComponent); err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to register discovery component")
	}

	kConsumer, err := kafka.NewConsumer(cfg, registry)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to init kafka consumer")
	}

	lifecycleSvc := services.NewLifecycleService(cfg, lcProducer)

	// wire
	repo := repositories.NewTopologyRepository(pool)
	svc := services.NewTopologyService(repo)

	app := fiber.New(fiber.Config{
		// optional: tune body limits, read/write timeouts are handled by env values for HTTP server if you run behind a reverse proxy
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 15,
	})

	th := handlers.NewTopologyHandler(svc)
	deps := http.ServerDeps{
		APIKey:         cfg.APIKey,
		TopologyWindow: cfg.TopologyWindow,
		TopologyDrift:  cfg.TopologyDrift,
	}

	http.New(app, deps, th)
	deps.RegisterRoutes(app, th)

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	go func() {
		if err := kConsumer.Run(runCtx); err != nil {
			logger.GetLogger().WithError(err).Error("kafka consumer stopped")
		}
	}()
	lifecycleSvc.Start(runCtx)

	err = (&deps).Start(app, *cfg, *pool)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to start http server")
	}

	runCancel()
	_ = app.Shutdown()
	_ = kProducer.Close()
	_ = kConsumer.Close()
}
