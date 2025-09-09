package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/router-architects/network-topology-service/adapters/postgres"
	"github.com/router-architects/network-topology-service/internal/config"
	"github.com/router-architects/network-topology-service/internal/http"
	"github.com/router-architects/network-topology-service/internal/http/handlers"
	"github.com/router-architects/network-topology-service/internal/logger"
	"github.com/router-architects/network-topology-service/internal/repositories"
	"github.com/router-architects/network-topology-service/internal/services"
)

type logrusAdapter struct{ *logrus.Entry }

func (l logrusAdapter) Trace(args ...interface{})         { l.Entry.Trace(args...) }
func (l logrusAdapter) Debug(args ...interface{})         { l.Entry.Debug(args...) }
func (l logrusAdapter) Info(args ...interface{})          { l.Entry.Info(args...) }
func (l logrusAdapter) Warn(args ...interface{})          { l.Entry.Warn(args...) }
func (l logrusAdapter) Error(args ...interface{})         { l.Entry.Error(args...) }
func (l logrusAdapter) Fatal(args ...interface{})         { l.Entry.Fatal(args...) }
func (l logrusAdapter) Tracef(f string, a ...interface{}) { l.Entry.Tracef(f, a...) }
func (l logrusAdapter) Debugf(f string, a ...interface{}) { l.Entry.Debugf(f, a...) }
func (l logrusAdapter) Infof(f string, a ...interface{})  { l.Entry.Infof(f, a...) }
func (l logrusAdapter) Warnf(f string, a ...interface{})  { l.Entry.Warnf(f, a...) }
func (l logrusAdapter) Errorf(f string, a ...interface{}) { l.Entry.Errorf(f, a...) }
func (l logrusAdapter) Fatalf(f string, a ...interface{}) { l.Entry.Fatalf(f, a...) }
func (l logrusAdapter) WithFields(fields logger.Fields) logger.Logger {
	return logrusAdapter{l.Entry.WithFields(logrus.Fields(fields))}
}
func (l logrusAdapter) WithField(key string, value interface{}) logger.Logger {
	return logrusAdapter{l.Entry.WithField(key, value)}
}
func (l logrusAdapter) WithError(err error) logger.Logger {
	return logrusAdapter{l.Entry.WithError(err)}
}

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
	logger.SetLogger(logrusAdapter{log.WithField("app", cfg.AppName)})

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

	// wire
	repo := repositories.NewTopologyRepository(pool)
	svc := services.NewTopologyService(repo, logger.GetLogger())

	app := fiber.New(fiber.Config{
		// optional: tune body limits, read/write timeouts are handled by env values for HTTP server if you run behind a reverse proxy
	})

	th := handlers.NewTopologyHandler(svc)
	http.New(app, http.ServerDeps{
		APIKey:         cfg.APIKey,
		TopologyWindow: cfg.TopologyWindow,
		TopologyDrift:  cfg.TopologyDrift,
	}, th)

	// health
	app.Get("/livez", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/readyz", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	// ---------- TLS setup ----------
	crt := cfg.TLS_CERT
	key := cfg.TLS_KEY
	if crt == "" || key == "" {
		log.Fatalf("TLS_CERT and TLS_KEY must be set")
	}

	if _, err := os.Stat(crt); err != nil {
		log.Fatalf("TLS cert not found/readable: %s (%v)", crt, err)
	}
	if _, err := os.Stat(key); err != nil {
		log.Fatalf("TLS key not found/readable: %s (%v)", key, err)
	}

	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Fatalf("Failed to load X509 key pair (cert=%s key=%s): %v", crt, key, err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	port := cfg.HTTPPort
	if port == 0 {
		if ep := os.Getenv("HTTP_PORT"); ep != "" {
			if v, convErr := strconv.Atoi(ep); convErr == nil {
				port = v
			}
		}
		if port == 0 {
			port = 8443
		}
	}
	addr := fmt.Sprintf(":%d", port)

	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		log.Fatalf("failed to start TLS listener on %s: %v", addr, err)
	}

	// ---------- serve + graceful shutdown ----------
	log.WithField("addr", addr).Info("Listening (TLS)")

	// Run the Fiber server in its own goroutine
	go func() {
		if err := app.Listener(ln); err != nil {
			logger.GetLogger().WithError(err).Error("fiber listener stopped")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Stop accepting new connections and shut down Fiber
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	if err := app.Shutdown(); err != nil {
		logger.GetLogger().WithError(err).Error("fiber shutdown error")
	}

	_ = ln.Close()

	// Close DB pool
	pool.Close()

	<-shutdownCtx.Done()
	log.Info("Shutdown complete")
}
