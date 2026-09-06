# 药房管理系统 Makefile（Linux / Git Bash）
.PHONY: build run vet fmt fmt-check lint test test-integration swag ci db-migrate db-up db-down docker-build docker-up docker-down docker-logs

APP := bin/yaofang

# 构建时注入版本信息（/version 端点与启动日志可见）
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDTIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X yaofang/internal/version.Version=$(VERSION) \
	-X yaofang/internal/version.Commit=$(COMMIT) \
	-X yaofang/internal/version.BuildTime=$(BUILDTIME)

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) ./cmd/server

# 交叉编译 Linux amd64（无 Docker 裸机部署用）
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/yaofang-linux-amd64 ./cmd/server

run:
	go run ./cmd/server

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@echo "检查未格式化文件..."
	@test -z "$$(gofmt -l .)" || (echo "以下文件未格式化:"; gofmt -l .; exit 1)

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "请先安装 golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

test:
	go test ./...

# 集成测试（需 PostgreSQL；本地可先 make db-up 用 Docker 起库并迁移）
test-integration:
	go test -tags=integration ./internal/service/

# 本地集成测试环境：Docker 起 PG + 执行迁移（需 Docker Compose）
db-up:
	docker compose up -d db
	docker compose run --rm migrate

db-down:
	docker compose down

# ── 容器化部署 ──
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDTIME=$(BUILDTIME) -t yaofang-api:$(VERSION) .
	docker build -t yaofang-web:$(VERSION) pharmacy-web
	docker build -f deploy/Dockerfile.migrate -t yaofang-migrate:$(VERSION) .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f --tail=200 api web

# 日志聚合栈（Loki + Promtail + Grafana）
logging-up:
	docker compose -f compose.logging.yml up -d

logging-down:
	docker compose -f compose.logging.yml down

swag:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 执行全部迁移脚本（需 PGPASSWORD/psql 可用）
db-migrate:
	@for f in migrations/*.up.sql; do echo "== $$f"; psql -h $${YF_DB_HOST:-localhost} -U $${YF_DB_USER:-yaofang} -d $${YF_DB_NAME:-yaofang} -v ON_ERROR_STOP=1 -f "$$f" || exit 1; done

# CI 门槛：静态检查 + 格式 + 单元 + 集成
ci: vet fmt-check test
	@echo "=== 集成测试（需 PostgreSQL）==="
	go test -tags=integration ./internal/service/
	@echo "=== CI 门槛通过 ==="
