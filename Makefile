# 药房管理系统 Makefile（Linux / Git Bash）
.PHONY: build run vet fmt fmt-check lint test test-integration swag ci db-migrate

APP := bin/yaofang

build:
	go build -o $(APP) ./cmd/server

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

test-integration:
	go test -tags=integration ./internal/service/

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
