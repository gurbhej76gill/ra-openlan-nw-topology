#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${1:-network-topology-svc}"

mkdir -p "$ROOT_DIR"/{adapters/postgres,cmd,internal/{apperrors,config,http/{handlers,middlewares},logger,models,repositories,services,utils},scripts}
touch "$ROOT_DIR"/{.gitignore,.dockerignore,Dockerfile,Jenkinsfile,README.md,docker-compose.yaml,go.mod}
touch "$ROOT_DIR"/cmd/main.go

# internal
touch "$ROOT_DIR"/internal/apperrors/errors.go
touch "$ROOT_DIR"/internal/config/config.go
touch "$ROOT_DIR"/internal/http/server.go
touch "$ROOT_DIR"/internal/http/routes.go
touch "$ROOT_DIR"/internal/http/handlers/topology.go
touch "$ROOT_DIR"/internal/http/middlewares/auth.go
touch "$ROOT_DIR"/internal/http/middlewares/request_logger.go
touch "$ROOT_DIR"/internal/logger/logger.go
touch "$ROOT_DIR"/internal/models/timepoint_row.go
touch "$ROOT_DIR"/internal/models/topology.go
touch "$ROOT_DIR"/internal/repositories/topology_repo.go
touch "$ROOT_DIR"/internal/services/topology_service.go
touch "$ROOT_DIR"/internal/utils/time.go

# adapters
touch "$ROOT_DIR"/adapters/postgres/pgx.go

echo "Scaffold created at: $ROOT_DIR"

