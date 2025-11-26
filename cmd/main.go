package main

import (
	"context"
	"crypto/x509"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/client"
	"github.com/sirupsen/logrus"

	serviceclient "github.com/router-architects/ra-openlan-nw-topology/adapters/httpclient"
	kafkaadapter "github.com/router-architects/ra-openlan-nw-topology/adapters/kafka"
	"github.com/router-architects/ra-openlan-nw-topology/adapters/logger"
	"github.com/router-architects/ra-openlan-nw-topology/internal/api"
	"github.com/router-architects/ra-openlan-nw-topology/internal/api/handlers"
	"github.com/router-architects/ra-openlan-nw-topology/internal/config"
	"github.com/router-architects/ra-openlan-nw-topology/internal/gateway/analytics"
	"github.com/router-architects/ra-openlan-nw-topology/internal/gateway/security"
	"github.com/router-architects/ra-openlan-nw-topology/internal/services"
	discoverycomponent "github.com/router-architects/ra-openlan-nw-topology/internal/services/discovery"
	"github.com/router-architects/ra-openlan-nw-topology/internal/services/lifecycle"
	"github.com/router-architects/ra-openlan-nw-topology/internal/services/discovery"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logrus.New()
	log.SetOutput(os.Stdout)

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
	if cfg.LogJSON {
		log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	} else {
		log.SetFormatter(&logger.LegacyFormatter{TimestampFormat: "2006-01-02 15:04:05.000"})
	}
	entry := logrus.NewEntry(log)
	logger.SetLogger(logger.LogrusAdapter{Entry: entry})

	svcDiscoveryStore := discovery.NewDiscoveryStore()

	tokenValidationClient := client.New()
	tokenValidationClient.SetTimeout(5 * time.Second)
	if cfg.TokenValidationCACert != "" {
		pemBytes, err := os.ReadFile(cfg.TokenValidationCACert)
		if err != nil {
			logger.GetLogger().WithError(err).Fatal("failed to read token validation CA cert")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			logger.GetLogger().Fatal("failed to parse token validation CA cert")
		}
		tokenValidationClient.TLSConfig().RootCAs = pool
	}

	OpenAPIRequestClient := serviceclient.NewOpenApiRequest(
		tokenValidationClient,
		serviceclient.OpenAPIRequestConfig{
			Timeout: 3 * time.Second,
		},
	)

	tokenValidator := security.NewTokenValidator(
		OpenAPIRequestClient,
		svcDiscoveryStore,
	)
	timepointClient := analytics.NewTimepointClient(OpenAPIRequestClient, svcDiscoveryStore)

	cmdProducer, err := kafkaadapter.NewProducerForTopic(cfg, cfg.KafkaTopicLifecycle)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to init kafka producer")
	}

	lifecycleProducer, err := kafkaadapter.NewProducerForTopic(cfg, cfg.KafkaTopicLifecycle) // for lifecycle events
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to init kafka producer (lifecycle)")
	}

	handlerRegistry := kafkaadapter.NewRegistry()
	discoveryComponent, err := discoverycomponent.NewDiscoveryComponent(cfg.KafkaTopicCmd, handlerRegistry, svcDiscoveryStore, 100)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to create discovery component")
	}

	consumer, err := kafkaadapter.NewConsumer(cfg, handlerRegistry)
	if err != nil {
		logger.GetLogger().WithError(err).Warn("failed to init kafka consumer")
	}

	lifecycleService := lifecycle.NewLifecycleService(cfg, lifecycleProducer)

	svc := services.NewTopologyService(timepointClient)

	app := fiber.New(fiber.Config{
		// optional: tune body limits, read/write timeouts are handled by env values for HTTP server if you run behind a reverse proxy
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 15,
	})

	th := handlers.NewTopologyHandler(svc)
	deps := api.ServerDeps{
		APIKey:         cfg.APIKey,
		TokenValidator: tokenValidator,
	}

	api.New(app, deps, th)
	deps.RegisterRoutes(app, th)

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	if consumer != nil {
		go func() {
			if err := consumer.Run(runCtx); err != nil {
				logger.GetLogger().WithError(err).Error("kafka consumer stopped")
			}
		}()
	}

	go discoveryComponent.Run(runCtx)

	lifecycleService.Start(runCtx)

	err = (&deps).Start(app, *cfg)
	if err != nil {
		logger.GetLogger().WithError(err).Fatal("failed to start http server")
	}

	runCancel()
	_ = app.Shutdown()
	_ = cmdProducer.Close()
	if consumer != nil {
		_ = consumer.Close()
	}
}
