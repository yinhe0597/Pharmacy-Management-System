# 药房管理系统 Makefile（Linux / Git Bash）
.PHONY: build run vet fmt fmt-check lint test test-integration cover swag ci env-init db-migrate db-backup db-up db-down docker-build docker-up docker-down docker-logs

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
	go test -tags=integration -count=1 ./internal/service/ ./internal/handler/

# 覆盖率门槛（domain≥85% / service≥35% / repository≥20% / handler≥10%；集成口径需数据库）
cover:
	bash scripts/check_coverage.sh

# 本地开发环境变量：生成含随机密钥的 .env（已存在则跳过）。
# compose 现已移除弱默认值（YF_DATABASE_PASSWORD / YF_AUTH_JWT_SECRET 必填），
# 本地起库/起全栈前先执行本目标即可，无需手工设置环境变量。
env-init:
	@if [ -f .env ]; then \
	  echo ".env 已存在，跳过生成（如需重置请先删除 .env）"; \
	else \
	  if command -v openssl >/dev/null 2>&1; then \
	    jwt=$$(openssl rand -base64 48 | tr -d '\r\n'); \
	    db=$$(openssl rand -hex 24 | tr -d '\r\n'); \
	  else \
	    jwt=$$(head -c 48 /dev/urandom | base64 | tr -d '\r\n'); \
	    db=$$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \r\n'); \
	  fi; \
	  if [ $${#jwt} -lt 32 ]; then echo "错误：JWT 密钥生成失败（长度不足 32）" >&2; exit 1; fi; \
	  umask 077; \
	  printf '# 本地开发环境变量（由 make env-init 生成，含密钥，勿提交版本库）\nYF_AUTH_JWT_SECRET=%s\nYF_DATABASE_PASSWORD=%s\n' "$$jwt" "$$db" > .env; \
	  echo "已生成 .env（随机 JWT 密钥 $${#jwt} 字符 + 随机数据库口令）"; \
	fi

# 本地集成测试环境：Docker 起 PG + 执行迁移（需 Docker Compose）
db-up: env-init
	docker compose up -d db
	docker compose run --rm migrate

db-down:
	docker compose down

# 数据库备份（pg_dump -Fc + 保留策略；支持本机 psql 或 docker compose 回退）
db-backup:
	bash scripts/backup_db.sh

# ── 容器化部署 ──
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDTIME=$(BUILDTIME) -t yaofang-api:$(VERSION) .
	docker build -t yaofang-web:$(VERSION) pharmacy-web
	docker build -f deploy/Dockerfile.migrate -t yaofang-migrate:$(VERSION) .

docker-up: env-init
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

# 执行数据库迁移（增量 + 版本表；需 psql 可用，连接参数经 YF_DB_* 覆盖）
# 历史库首次升级：make db-migrate YF_MIGRATE_BASELINE_TO=000032
# 口令必须一并导出：scripts/migrate.sh 只读 PG*/MIGRATIONS_DIR，不读任何 YF_ 口令变量，
# 此前漏设 PGPASSWORD 会让对有口令的库执行 make db-migrate 直接 fe_sendauth 失败。
db-migrate:
	PGHOST=$${YF_DB_HOST:-localhost} PGUSER=$${YF_DB_USER:-yaofang} PGDATABASE=$${YF_DB_NAME:-yaofang} \
	  PGPASSWORD=$${YF_DB_PASSWORD:-$${YF_DATABASE_PASSWORD:-$${PGPASSWORD:-}}} \
	  MIGRATIONS_DIR=migrations sh scripts/migrate.sh

# CI 门槛：静态检查 + 格式 + 单元 + 集成
ci: vet fmt-check test
	@echo "=== 集成测试（需 PostgreSQL）==="
	go test -tags=integration ./internal/service/
	@echo "=== CI 门槛通过 ==="
