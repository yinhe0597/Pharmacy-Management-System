# 药房管理系统

Go + PostgreSQL 实现的药房进销存与处方调配后端。模块化单体，一期独立运行，二期通过预留接口扩展诊疗模块。

- 开发文档：见 [docs/README.md](docs/README.md)
- API 文档：服务启动后访问 `http://localhost:8080/swagger/index.html`

## 技术栈

Go 1.22+ / Gin / GORM / PostgreSQL 14+ / golang-migrate（SQL 迁移）/ JWT / robfig-cron

## 快速开始

### 1. 环境要求

- Go 1.22+
- PostgreSQL 14+

### 2. 初始化数据库

```bash
# 创建角色与数据库（postgres 超级用户）
psql -U postgres -h localhost -c "CREATE ROLE yaofang LOGIN PASSWORD 'yaofang123';"
psql -U postgres -h localhost -c "CREATE DATABASE yaofang OWNER yaofang;"

# 执行迁移与种子（migrations/ 目录）
psql -U postgres -h localhost -d yaofang -f migrations/000001_init.up.sql
psql -U postgres -h localhost -d yaofang -f migrations/000002_seed.up.sql

# 若表由其他角色创建，需授予应用角色权限（否则改 OWNER）
# ALTER TABLE ... OWNER TO yaofang; ALTER SEQUENCE ... OWNER TO yaofang;
```

### 3. 配置

```bash
cp configs/config.example.yaml configs/config.yaml   # 首次使用
```
修改数据库密码与 JWT 密钥；也可用环境变量 `YF_` 前缀覆盖（如 `YF_DATABASE_PASSWORD`、`YF_AUTH_JWT_SECRET`）。
`configs/config.yaml` 含本地凭据，已加入 `.gitignore` 不入库。

### 4. 启动

```bash
go run ./cmd/server
# 或构建
go build -o bin/yaofang.exe ./cmd/server && ./bin/yaofang.exe
```

默认管理员：`admin / admin123`（生产环境务必修改）。

### 5. 测试

```bash
go test ./...                          # 单元测试（domain/money/rule）
go test -tags=integration ./internal/service/   # 集成测试（需 PostgreSQL，走全链路）
python scripts/smoke_test.py           # HTTP 冒烟测试（需服务已启动）
```

## 业务能力

药品主数据（一药多规/一品多商/分类分级/配伍禁忌）· 供应商与采购（质检收货绑定批次）· 库存（批号效期追踪、FEFO 发药、预占/实扣/释放、拆零、调拨、盘点、效期与上下限预警）· 处方（录入→审核→调配→发药→退药全状态机）· 特殊药品「五专」· 药学服务 · 报表。

## 二期预留

患者/计价/库存能力通过 `internal/service/port` 接口抽象，详见 [docs/05-二期预留接口设计.md](docs/05-二期预留接口设计.md)。
