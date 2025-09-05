#!/usr/bin/env bash
set -e

# Root
mkdir -p network-topology-service
cd network-topology-service

# Go module init placeholder
touch go.mod go.sum

# Docker & CI
touch Dockerfile docker-compose.yaml Jenkinsfile README.md .gitignore .dockerignore

# Command entrypoint
mkdir -p cmd
touch cmd/main.go

# Adapters
mkdir -p adapters/postgres
touch adapters/postgres/pgx.go

# Internal
mkdir -p internal/{apperrors,config,http/handlers,http/middlewares,logger,models,repositories,services,utils}

# Apperrors
touch internal/apperrors/errors.go

# Config
touch internal/config/config.go

# HTTP
touch internal/http/routes.go
touch internal/http/server.go
touch internal/http/handlers/topology.go
touch internal/http/middlewares/auth.go
touch internal/http/middlewares/request_logger.go

# Logger
touch internal/logger/logger.go

# Models
touch internal/models/timepoint_row.go
touch internal/models/topology.go

# Repositories
touch internal/repositories/topology_repo.go

# Services
touch internal/services/topology_service.go

# Utils
touch internal/utils/time.go

echo "Project skeleton created successfully."

