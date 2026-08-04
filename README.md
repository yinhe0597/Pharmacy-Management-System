# 药房管理系统

Go + PostgreSQL 实现的药房进销存与处方调配后端。模块化单体，一期独立运行，二期通过预留接口扩展诊疗模块。

- 版本与变更：见 [CHANGELOG.md](CHANGELOG.md)
- 开发文档：见 [docs/README.md](docs/README.md)
- API 文档：服务启动后访问 `http://localhost:8080/swagger/index.html`

## 技术栈

Go 1.22+ / Gin / GORM / PostgreSQL 14+ / golang-migrate（SQL 迁移）/ JWT / robfig-cron

## 功能概览

- **药房物品**：药品+耗材统一管理（item_type），批号效期/FEFO/拆零/配伍禁忌
- **药物相互作用引擎**：4 策略分层匹配（显式→成分→分类→标签）、患者个体化禁忌、37 条种子规则
- **采购**：供应商、采购单状态机、质检收货（批次/效期绑定）
- **库存**：FEFO 发药、预占/实扣/释放、调拨、盘点、预警、**领用出库**（内部消耗不计费）
- **拆零**：按盒/按片拆零、混合发药（LDU 精确计价）、自动拆零
- **处方**：录入→药师审核（pass/reject/return）→调配→发药→退药全状态机
- **诊疗项目**：手法复位/注射等不入药房库存，独立计价 → 计费记录统一入口
- **特殊药品「五专」**、**药学服务**、**报表**
- **用户角色**：7 种角色分级权限，调配+核对由医生/药师兼任
- **参考数据**：ICD-10 诊断编码（1,586 条）+ 国家集采药品目录（392 品种）+ 医保药品目录（3,313条）+ **非医保常用药品（362种）** + 医用耗材目录（141类）
- **二期预留**：`port` 三接口 + 契约测试

## 快速开始

### 1. 环境要求

- Go 1.22+
- PostgreSQL 14+

### 2. 初始化数据库

```bash
# 创建角色与数据库（postgres 超级用户）
psql -U postgres -h localhost -c "CREATE ROLE yaofang LOGIN PASSWORD 'yaofang123';"
psql -U postgres -h localhost -c "CREATE DATABASE yaofang OWNER yaofang;"

# 依次执行全部迁移与种子（migrations/NNNNNN_*.up.sql，共 19 个版本）
for f in migrations/*.up.sql; do
  echo "== $f"
  psql -U postgres -h localhost -d yaofang -v ON_ERROR_STOP=1 -f "$f"
done
# 等价：make db-migrate（Linux/Git Bash，可配置 YF_DB_HOST/USER/NAME）

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

默认管理员：`admin / admin123`。其他种子用户：`doctor`、`nurse`、`pharmacy_chief`、`pharmacist`、`buyer`、`finance`（密码均 `admin123`，生产环境务必修改）。

### 5. 测试

```bash
go test ./...                          # 单元测试（domain/money/rule）
go test -tags=integration ./internal/service/   # 集成测试（需 PostgreSQL，走全链路）
python scripts/smoke_test.py           # HTTP 冒烟测试（需服务已启动）
```

## 项目状态

一期已全部交付，v1.3.0 新增 5 张参考数据表（ICD-10 + 集采 + 医保 + 非医保 + 耗材），共计 **6,794 条**种子数据。
功能完整、测试全绿（38 单元 + 24 集成）。
详见 [CHANGELOG.md](CHANGELOG.md)。

## 二期预留

患者/计价/库存能力通过 `internal/service/port` 接口抽象，详见 [docs/05-二期预留接口设计.md](docs/05-二期预留接口设计.md)。
