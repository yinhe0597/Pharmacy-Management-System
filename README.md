<div align="center">

# 💊 药房管理系统

**版本 v1.5.0**（软件著作权登记版本）

**Go + Vue3 全栈药房管理系统——进销存 · 处方调配 · 二期就诊 · Docker 一键部署**

模块化单体架构 · 全栈交付（Vue3 + ECharts）· 容器化一键部署 · 二期诊疗无缝扩展

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Vue3](https://img.shields.io/badge/Vue3-4FC08D?style=flat-square&logo=vuedotjs&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-Web%20Framework-00ADD8?style=flat-square&logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-336791?style=flat-square&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-一键部署-2496ED?style=flat-square&logo=docker&logoColor=white)
![GORM](https://img.shields.io/badge/GORM-ORM-2e8b57?style=flat-square)
![Swagger](https://img.shields.io/badge/Swagger-REST%20API-85EA2D?style=flat-square&logo=swagger&logoColor=black)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen?style=flat-square)
![Lint](https://img.shields.io/badge/lint-golangci--lint-2e8b57?style=flat-square)

</div>

---

## 📖 目录

- [✨ 功能特性](#功能特性)
- [🛠️ 技术栈](#技术栈)
- [🚀 快速开始](#快速开始)
- [🏗️ 项目结构](#项目结构)
- [📡 API 文档](#api-文档)
- [📊 项目状态与路线图](#项目状态与路线图)
- [🔭 二期规划](#二期规划)
- [📄 文档与变更](#文档与变更)

---

## ✨ 功能特性

### 💊 药房物品主数据
> 药品 + 耗材统一管理（`item_type`），一药多规/一品多商，批号效期全程追踪，FEFO 先进先出，启停用/冻结。

| 🏷️ 模块 | 📋 能力 |
|---------|--------|
| 🧬 药物相互作用引擎 | 4 策略分层匹配（显式 → 成分 → 分类 → 标签）、患者个体化禁忌（年龄/妊娠/哺乳/过敏史）、**37 条种子规则** |
| 📦 采购管理 | 供应商维护、采购单状态机、质检收货（批次/效期绑定）、防超收 |
| 🗃️ 库存管理 | FEFO 发药、预占/实扣/释放（行锁 + 条件更新双保险）、调拨、盘点、预警处置闭环、领用出库、**耗材领用/补发登记单**（不计费） |
| ✂️ 拆零管理 | 按盒/按片拆零、混合发药（LDU 精确计价）、自动拆零、**拆零操作单（麻精双人复核）**、拆零统计（量/损耗/毛利） |
| 📝 处方全流程 | 录入 → 药师审核（pass/reject/return）→ 调配 → 发药 → 退药全状态机；审计日志可追溯；**结构化诊断编码（ICD-10）**；**医嘱核对（verify-order）**；列表支持按患者/类型筛选 |
| 🛡️ 特殊药品「五专」 | 专用处方、双人核对（**调配 ≠ 核对强制分离**）、专账联动、空安瓿回收 |
| 🧑‍⚕️ 患者档案 | 建档 / 过敏史 / 哺乳史 / **用药史**，审核自动带入过敏史禁忌（3010）与哺乳期慎用（3011）；**开方自动回填患者信息**；处方详情聚合档案与过敏史 |
| 💰 计费闭环 | 药品/耗材/诊疗项目统一计费入口，**从已发药处方一键生成**，退药自动冲正（同事务原子化）；**红冲（void）**、7 维筛选、患者费用聚合报表 |
| 🩺 诊疗项目 | 手法复位/注射/检查等不入药房库存，独立计价；**目录启停用** |
| 👥 角色权限 | 8 种角色分级权限（**跟诊护士**负责患者管理/核对医嘱，**药房护士**负责耗材补发/处方执行/辅助核对收费），**RBAC 按角色矩阵强制**（计费/患者隐私按角色收窄），调配+核对由医生/药师兼任 |
| 📚 参考数据 | ICD-10（1,586 条）+ 集采目录（392 品种）+ 医保目录（3,313 条）+ 非医保（362 种）+ 耗材（141 类），只读查询与目录匹配 API |
| 🏥 二期就诊模块 | 挂号/分诊（就诊状态机 waiting→visiting→finished/cancelled）、病历（主诉/现病史/体征/多诊断 ICD-10）、**合并结算**（药费+诊疗项目费聚合 CalculateBill、收费/退费状态机）、处方关联就诊（source 扩展 outpatient/inpatient/refill）、**S7 结算报表**（就诊量/收入构成/诊断分布） |
| ⚙️ 系统设置 | 管理员自定义**默认诊费**（挂号费/诊查费，合并结算自动带入）、**默认收货/发药库房**、**操作日志保留天数**、用户账号管理（新建/编辑/**重置密码**）、全量操作日志审计（用户/动作/时间筛选） |
| 🔔 站内通知 | 调度器预警触达（过期锁定 critical 广播/每日预警摘要 warning，同日幂等），顶栏铃铛未读轮询 + 下拉已读（已读按用户隔离） |
| 🔌 二期预留 | `port` 三接口（患者/库存/计价）+ 契约测试固化 |

---

## 🛠️ 技术栈

| 层 | 选型 | 说明 |
|----|------|------|
| 🗣️ 语言 | **Go 1.26+** | 静态类型、高并发、编译部署简单 |
| 🌐 Web 框架 | **Gin** | 高性能、生态成熟 |
| 🗄️ ORM | **GORM** | 事务、行级锁（`clause.Locking`）、软删除 |
| 🐘 数据库 | **PostgreSQL 14+** | 行级锁、报表聚合，库存正确性第一 |
| 📦 迁移 | **golang-migrate（SQL）** | 生产级可控，SQL 为唯一事实来源 |
| 🔐 鉴权 | **JWT + bcrypt** | 记录操作人，支撑双人核对审计 |
| 📄 API 文档 | **swaggo/swag** | 注解即文档，随代码生成 |
| ⏰ 定时任务 | **robfig/cron/v3** | 效期预警、过期锁定、库存下限预警 |
| 💰 金额 | **int64（分）** | 全整数运算，杜绝浮点误差 |

---

## 🚀 快速开始

### ⚡ 3 分钟跑起来（Docker 一键，推荐）

```bash
# Windows：双击 start.bat  ｜  Linux/macOS：./start.sh
# 首次运行自动生成 .env（CSPRNG 随机 JWT 密钥 ≥32 位 + 随机数据库口令），构建并等待就绪
# 访问 http://localhost 或 http://<服务器内网IP>，默认账号 admin / admin123（登录后请立即改密）
```

> 本地开发（无 Docker）：起 PostgreSQL → 执行迁移 → `go run ./cmd/server`，见下文分步说明。

### ⚙️ 环境要求

| 依赖 | 版本 |
|------|------|
| Go | ≥ 1.26.6 |
| PostgreSQL | ≥ 14 |
| Docker Compose（可选） | 一键起库 / 全栈部署 |

### 🐳 容器化一键部署（可选，需 Docker）

```bash
# 需显式提供强口令与随机密钥（dev/prod 编排均已移除默认弱口令）
YF_DATABASE_PASSWORD='<强口令>' YF_AUTH_JWT_SECRET='<≥32 位随机串>' docker compose up -d --build
# 前端 http://localhost:8088 ，API 健康检查 http://localhost:8080/readyz
# 或直接使用生产一键包：Windows 双击 start.bat / Linux ./start.sh（自动生成 .env 随机密钥与口令）
```

> `docker compose` 现已**移除弱默认值**，要求显式提供 `YF_DATABASE_PASSWORD` 与 `YF_AUTH_JWT_SECRET`（缺失时编排直接报错拒绝启动）。
> 本地开发不必每次内联口令：`cp .env.example .env` 后填两项变量即可（`.env` 已被 gitignore）；若使用 make，可用 `make env-init` 生成 `.env`（自动填入随机强口令与 ≥32 位 JWT 密钥）。

无 Docker 的服务器可走 **裸机 systemd 部署**（`deploy/yaofang.service`），K8s 部署见 `deploy/k8s/`，完整说明见 [docs/10-部署运维.md](docs/10-部署运维.md)。

### 🗄️ 1️⃣ 初始化数据库

```bash
# 方式一：Docker 一键起库并执行迁移（推荐 🐳；首次会自动生成含随机密钥的 .env）
make db-up                                   # = make env-init + docker compose up -d db + 增量迁移

# 方式二：本机 PostgreSQL（postgres 超级用户；口令仅本地开发示例，生产用随机强口令）
psql -U postgres -h localhost -c "CREATE ROLE yaofang LOGIN PASSWORD 'yaofang123';"
psql -U postgres -h localhost -c "CREATE DATABASE yaofang OWNER yaofang;"

# 执行迁移（共 41 个版本）：增量 + 版本表，可安全重复执行
make db-migrate                              # = scripts/migrate.sh（PGHOST/PGUSER/PGDATABASE/YF_DB_* 可覆盖）
# 历史库（旧版「全量重放」方式初始化）首次升级需一次性指定基线：
#   YF_MIGRATE_BASELINE_TO=000032 make db-migrate
```

> ⚠️ 不要再手工 `for f in migrations/*.up.sql; psql -f "$f"` 全量重放：该方式只在全新库上成立，
> 对已初始化的库会因 `CREATE TABLE already exists` 失败并阻断后端启动。

### ⚙️ 2️⃣ 配置

```bash
cp configs/config.example.yaml configs/config.yaml   # 首次使用
```

修改数据库密码与 JWT 密钥；也可用环境变量 `YF_` 前缀覆盖（如 `YF_DATABASE_PASSWORD`、`YF_AUTH_JWT_SECRET`）。
`configs/config.yaml` 含本地凭据，已加入 `.gitignore` 不入库。🔒

### ▶️ 3️⃣ 启动

```bash
go run ./cmd/server
# 或构建
go build -o bin/yaofang.exe ./cmd/server && ./bin/yaofang.exe
```

> 👤 默认管理员：`admin / admin123`；其他种子用户：`doctor`、`clinic_nurse`（跟诊护士）、`pharmacy_nurse`（药房护士）、`pharmacy_chief`、`pharmacist`、`buyer`、`finance`（密码均 `admin123`，生产环境务必修改 ⚠️）
> 启动校验：JWT 密钥为空/占位值/<32 位将拒绝启动；`release` 模式下 CORS 为 `*` 会告警；Swagger 仅非 release 开放。

### 🚀 服务器一键部署（Docker 生产包）

无需安装 Go/Node/PostgreSQL，装好 Docker 即可：

```bash
# Windows：双击 start.bat        Linux：./start.sh
docker compose -f docker-compose.prod.yml up -d --build
```

- 首次运行自动生成 `.env`（**随机 JWT 密钥** + 数据库密码 + 端口），密钥不入库
- 自动执行全部迁移与种子数据（独立数据卷 `pgdata_prod`，不影响开发库）
- 架构：`Nginx(前端+API反代) → Go 后端 → PostgreSQL`，数据库端口不对外发布
- 访问：`http://localhost` 或 **`http://<服务器内网IP>`**（内网其它设备浏览器直接登录，同源反代无需 CORS）
- 运维：`docker compose -f docker-compose.prod.yml ps|logs -f|down`；数据备份即备份 `pgdata_prod` 卷
- ⚠️ 上线后请立即修改默认密码；已部署库升级时只需增量执行新迁移，勿全量重放

### 🌐 前端联调

- 后端已支持 **CORS 跨域**（`server.cors_allow_origins` 白名单，开发默认放行 `*`，生产限定域名；`release` 下使用 `*` 会告警）。
- 接口文档：`http://localhost:8080/swagger/index.html`（**仅非 release 模式开放**；含全部端点与 TS 类型生成来源 `docs/swagger.json`）。
- 对接约定与前端须知见 [docs/16-前端开发就绪评估与对接指南.md](docs/16-前端开发就绪评估与对接指南.md)。
- 前端技术选型、工程结构、页面规划与进度计划见 [docs/17-前端开发指南与进度规划.md](docs/17-前端开发指南与进度规划.md)。
- 前端工程 `pharmacy-web/`：P0-P6 全部完成（登录/RBAC/主数据/采购库存/患者处方核心流程含给药途径与分批配伍分组/计费/药学服务/特殊药品/报表/系统管理/CI），报表页已接入 **ECharts**（进销存汇总柱状图、效期分析饼图，**S7 三 Tab + CSV 导出**），详见 [pharmacy-web/README.md](pharmacy-web/README.md)。
- **单据打印**：处方签（处方详情）、收费结算单（收费台抽屉）、盘点单（库存盘点抽屉），新窗口精简版式一键打印。
- **生产级补全**：盘点管理全流程（新建冻结→开始→录实盘→差异调整→归档）、按片拆零（双人复核）、收货单质检确认入库、交互规则可视化维护（配伍/成分/分类/标签 + 药品成分）、供货关系绑定、麻精发药专册补录、个人中心（资料/改密）、库房管理、患者/诊疗项目/药学服务编辑删除；工作台统计卡与按权限快捷入口；患者/药品统一远程搜索选择器。

### ✅ 4️⃣ 测试

```bash
go test ./...                          # 单元测试（domain/service/middleware/config/pkg）
# 提示：frontend 依赖树内含示例 Go 包，`go test ./...` 会顺带扫描（无害）；
#      只想跑后端可执行 `go test ./cmd/... ./internal/...`
go test -tags=integration ./internal/service/   # 集成测试（需 PostgreSQL；先 make db-up）
go test -tags=integration ./internal/handler/   # HTTP 层测试（鉴权/RBAC/限速/审计/分页）
golangci-lint run ./...                # 静态检查（CI 门槛）
go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # 供应链漏洞扫描（当前 0 可达）
bash scripts/check_coverage.sh         # 覆盖率门槛：domain≥85% / service≥35% / repository≥20% / handler≥10%
python scripts/smoke_test.py           # HTTP 冒烟测试（需服务已启动）

# 前端（pharmacy-web/）
npm run type-check && npm run lint && npm run format:check && npm run test && npm run build
```

---

## 🏗️ 项目结构

```
yaofang/
├── cmd/server/            # 🚪 程序入口
├── internal/
│   ├── config/            # ⚙️ 配置加载（Viper + 环境变量覆盖）
│   ├── server/            # 🧩 Gin 引擎、路由注册、依赖装配
│   ├── model/             # 📊 GORM 数据模型（一张表一个文件）
│   ├── domain/            # 🧠 纯领域模型：状态机、金额、库存规则、交互引擎
│   ├── repository/        # 🗄️ 仓储实现（按模块分包）
│   ├── service/           # ⚡ 领域服务（业务规则、事务编排）
│   │   └── port/          # 🔌 二期预留接口（IPatientService/IStockService/IPricingService）
│   ├── handler/           # 🌐 HTTP 处理器 + 路由分组（RBAC）
│   ├── middleware/        # 🛡️ JWT、日志、恢复、请求ID、写审计、安全头
│   ├── scheduler/         # ⏰ 定时任务
│   └── pkg/               # 🧰 通用组件（errs/money/pagination/auth）
├── migrations/            # 📦 golang-migrate SQL 迁移（41 个版本，scripts/migrate.sh 增量执行）
├── configs/               # ⚙️ 配置样例
├── deploy/                # 🚢 部署资产：systemd 单元、K8s manifests、日志聚合配置
├── docs/                  # 📚 开发文档（28 篇编号文档，00–28）
├── scripts/               # 🔧 运维/构建/覆盖率脚本
├── Dockerfile             # 🐳 后端镜像（多阶段构建）
└── docker-compose.yml     # 🐳 全栈编排（db + migrate + api + web）
```

---

## 📡 API 文档

服务启动后访问：**http://localhost:8080/swagger/index.html**

接口统一前缀 `/api/v1`，响应信封 `{code, message, data}`，金额一律整数「分」。

健康检查（无鉴权，供负载均衡 / K8s 探针）：`GET /healthz`（存活）、`GET /readyz`（就绪，含数据库连通性）、`GET /version`（构建版本）。

| 🗂️ 模块 | 亮点端点 |
|--------|---------|
| 🔐 认证 | `POST /auth/login` · 用户管理（admin） |
| 💊 药品 | 主数据 CRUD · 配伍禁忌 · 交互规则（成分/分类/标签） |
| 📦 采购 | 采购计划建议 · 采购单 · 收货质检入库 |
| 🗃️ 库存 | 调拨 · 拆零/按片拆零 · 拆零操作单 · 盘点 · 预警处置 · 领用/补发登记单 |
| 📝 处方 | 录入 → 审核 → 调配 → 发药 → 退药 → 作废 |
| 🧑‍⚕️ 患者 | 建档 / 过敏史 / **用药史** / 处方详情聚合档案 |
| 💰 计费 | 手工计费 · `from-prescription` 一键计费 · **红冲 void** · 7 维筛选 |
| 🩺 诊疗项目 | 目录 CRUD · **启停用 status** |
| 🏥 就诊（二期） | 挂号/分诊 · 接诊/结束/退号 · 病历（多诊断） · **合并结算（药费+诊疗费）** |
| 🔔 通知 | 我的通知 · 未读数 · 标记已读/全部已读 |
| 📚 参考数据 | 诊断编码 / 集采 / 医保 / 非医保 / 耗材搜索 · 目录匹配 |
| 📊 报表 | 进销存汇总 · 效期分析 · 特殊药品统计 · 调配工作量 · 拆零统计 · **患者费用聚合** · **就诊量/收入构成/诊断分布（S7）** · **CSV 导出** |

---

## 📊 项目状态与路线图

### ✅ 当前状态

| 里程碑 | 状态 |
|--------|------|
| 一期核心闭环（主数据/采购/库存/处方/拆零） | ✅ 已交付 |
| 药物相互作用引擎（4 策略 + 患者级检查） | ✅ 已交付 |
| 参考数据（ICD-10/集采/医保/耗材/非医保，5,794 条） | ✅ 已交付并接线 |
| RBAC 角色矩阵强制 / 麻精双人核对 | ✅ 已落地 |
| 患者档案 + 计费闭环 + 拆零操作单/统计 | ✅ 已交付 |
| 诊疗模块复审修复（docs/15） | ✅ 已交付 |
| 前端就绪（CORS + 对接指南 docs/16 + 开发指南/进度 docs/17） | ✅ 已就绪 |
| CI 门槛（golangci-lint + 分层覆盖率） | ✅ 已落地 |
| 本地全链路联调实测（PG16 迁移 + HTTP 冒烟 + 集成测试） | ✅ 全绿（**实测 PG16**；PG14/15 未实测） |
| 前端 ECharts 报表增强（进销存汇总/效期分析） | ✅ 已交付 |
| 二期就诊模块 S1-S5（就诊/病历/合并结算后端 + 处方联动） | ✅ 已交付（联调全通） |
| 二期就诊模块 S6（前端就诊工作台 + 收费台） | ✅ 已交付 |
| 管理员能力（默认诊费配置 + 账号/密码管理 + 操作日志筛选） | ✅ 已交付 |
| 安全与正确性加固（登录限速/停用即时失效/重复红冲与重复计费防护/质检闭环/退药单位口径修复） | ✅ 已交付并实测 |
| 前端生产级补全（盘点全流程/收货质检/交互规则维护/个人中心/供货关系/工作台重构） | ✅ 已交付并端到端实测 |
| Docker 一键部署包（Nginx+后端+PG 全容器化，内网浏览器直访，自动迁移/随机密钥） | ✅ 已实测（镜像构建/迁移/登录/内网访问全通） |
| 部署运维完善（容器化 + 健康检查 + 日志聚合 + 连接池 + CI/CD） | ✅ 已交付 |
| 健康检查冲刺（安全加固 + 测试补齐 + 依赖升级 + 文档修正） | ✅ 已交付 |
| 第二轮审计修复（部署阻断/采购超收/并发入库/限速绕过/供应链漏洞 + 文档补全） | ✅ 已修复并回归 |
| 第三轮审计收尾（增量迁移/默认口令门禁/写操作审计覆盖/预警去重/测试隔离） | ✅ 已修复并实测 |
| 第四轮生产差异收口（处方 FK/网关限速与安全头/ECharts 按需/分层覆盖率） | ✅ 已修复 |
| 第五轮业务完善（S7 报表/CSV 导出/库房可配置/日志归档/站内通知/打印/测试守护） | ✅ 已交付（集成口径待另一环境实测） |
| 第六轮审计修复（Docker 构建双重阻断/质检门禁/账务口径统一/单号多副本/令牌吊销/效期口径/字段契约） | ✅ 已交付（见 docs/28） |
| 第六轮收尾（前后端字段契约守卫/预警行 DTO 修复/CI 触发分支修正/集成测试库隔离） | ✅ 已交付（见 docs/28 §十一） |
| 二期就诊模块规划（docs/20） | ✅ S1-S7 全部交付 |

> 迁移至 `000041`，共 **41 个版本**（`schema_migrations` 版本表驱动，增量执行、可重复运行）；
> 质量门禁：后端 `go build` / `go vet` / `go test` / `gofmt` / `golangci-lint` / `govulncheck` +
> 前端 `vue-tsc` / `eslint` / `prettier` / `vitest` / `build` 全绿 ✅
>
> **主干分支为 `master`**（Gitee `origin/master`）。两条流水线此前只监听 `main`，
> 导致推送 master 时 CI 整体不触发，已修正为 `[master, main]`。
> 本地跑集成测试请先 `make test-db-init`（建独立测试库 `yaofang_test`，不碰开发库）。

### 🧭 生产就绪检查清单

| 项 | 状态 | 说明 |
|----|------|------|
| 核心业务端到端（就诊→开方→发药→结算） | ✅ | HTTP 冒烟 43 项 + 集成场景 8 类（真实 PG） |
| 安全基线（注入/越权/弱口令/密钥） | ✅ | SQL 全参数化、RBAC 矩阵、弱密钥拒启、默认口令 release 拒启、**JWT 令牌可吊销（改密即失效）**、release 下 CORS 通配拒启、登录限速（后端 IP+用户名 + Nginx 网关）、可信代理白名单、CORS fail-closed + `Vary: Origin`、安全响应头 |
| 容器构建 | ✅ | `docker build` 实测通过（第六轮修复了 Go 版本不匹配与 `.dockerignore` 排除 `docs/` 两处阻断） |
| **多副本部署** | ✅ | 业务单号改号段表（`doc_segments`），5~20 副本下全局唯一（实测 20 分配器 × 10 单号零重复） |
| 账务口径 | ✅ | `charges`+`charge_items` 为唯一记账凭证；`charge_records` 为应收项目源，按 `visit_id` 精确归集（见 docs/28 §5.1） |
| 审计与脱敏 | ✅ | 全量写操作审计（`operation_logs`，`context.WithoutCancel` 防规避）+ 处方/库存领域审计 + 证件/手机号脱敏 |
| 供应链漏洞扫描 | ✅ | govulncheck 0 可达（Go 1.26.6）；`npm audit` 0 漏洞 |
| 结构化日志 + 聚合 | ✅ | slog text/json + Loki/Promtail/Grafana（`compose.logging.yml`；生产 compose 已显式设 `YF_LOG_FORMAT=json`） |
| 数据库备份 | ✅ | `make db-backup`（pg_dump -Fc + 保留策略 + 容器回退 + crontab，见 docs/10 §12） |
| 数据库迁移 | ✅ | `scripts/migrate.sh`：`schema_migrations` 版本表 + 增量执行 + 单文件单事务（可重复运行，至 000041） |
| 前端构建优化 | ✅ | Nginx gzip + 强缓存、路由懒加载、vendor 分包（vue/element-plus/echarts 按需/axios） |
| 压力测试（50+ 并发 <500ms） | 📋 | 待专项执行（见 docs/12 容量公式） |
| 备份恢复演练 | 📋 | 建议每季一次（恢复至临时库校验） |
| 合规性专业评估（等保/医疗数据） | 📋 | 建议引入第三方评估，系统侧控制已就位 |

### 🗺️ 路线图

- ✅ **P0** 正确性/合规/安全：RBAC、退回医生死路、双人核对、参考数据接线
- ✅ **P1** 功能补强：患者档案、计费闭环、目录匹配、拆零单/统计、预警闭环
- ✅ **P2** 工程化：lint/覆盖率门槛、docker-compose、测试补强
- ✅ **前端联调实测**：本地 PG16 全量迁移 + HTTP 冒烟（15 项检查）全通 + 集成测试全绿
- ✅ **二期就诊模块后端**：S1 迁移（visits/medical_records(+diagnoses)/charges(+items)）+ S2 就诊域 + S3 病历域 + S4 结算域（`CalculateBill` 合并计价）+ S5 处方联动（`visit_id`/`source`）；二期冒烟（18 项检查）全通
- ✅ **二期就诊模块前端**：S6 就诊工作台（挂号/接诊/退号/病历/结算）+ 收费台（明细/收费/退费）页面
- ✅ **安全与正确性加固**：登录限速/停用复查/JWT 密钥校验、重复红冲与重复计费防护、领用不吞预占、退药单位口径修复、质检闭环、结算单幂等（000031 唯一索引）、库存调拨/流水/预警前端补齐
- ✅ **部署运维**：Dockerfile（后端/前端/迁移）+ 全栈 compose + K8s manifests + 裸机 systemd +
  健康检查（/healthz /readyz /version）+ 结构化日志 + Loki 日志聚合栈 + 连接池配置说明 + CI/CD 流水线
- ✅ **健康检查冲刺**：生产禁弱 JWT 密钥启动 + 登录失败审计日志 + RequireRoles 安全断言 +
  x/crypto 升级 + config/sanitize/middleware 单测补齐 + 文档计数修正
- ✅ **第二轮审计修复**：Linux 一键部署密钥生成阻断修复；采购收货并发超收（原子条件更新 + 在途扣减）；
  入库并发唯一键冲突改 `ON CONFLICT` upsert；盘点差异口径修正；鉴权每请求复查角色；
  可信代理白名单（防 XFF 伪造绕过限速）；Swagger 生产关闭；prod 探针改 `/readyz`；
  Go 1.26.6 + quic-go v0.59.1（govulncheck 0 可达）；前端 vitest 单测接入 CI
- ✅ **第四轮生产差异收口**：处方 `patient_id` 外键（000034）；审计记登录名；Nginx 登录限速与安全头；ECharts 按需引入；分层覆盖率门槛 + HTTP 层集成测试
- ✅ **第五轮业务完善**：S7 结算报表三端点 + CSV 导出；收货/发药库房可配置（000035，退药溯源回补）；操作日志归档（000035，调度器每日搬运）；站内通知（000036，过期锁定/预警日报 + 顶栏铃铛）；处方签/结算单/盘点单打印；全链路版本统一 v1.5.0；集成测试清理清单守护
- ✅ **第六轮审计修复**（详见 [docs/28-第六轮审计修复报告.md](docs/28-第六轮审计修复报告.md)）：
  - **阻断**：`docker build` 双重阻断（Go 版本不匹配 + `.dockerignore` 排除 `docs/`）——此前任何容器化路径都必然失败
  - **合规**：收货质检门禁失效（`qc_result` 的 `default:1` 把「未质检」改写为「合格」，收货可零质检入库）；麻精 `days=0` 绕过限量
  - **正确性**：库存调拨负数量致源库房虚增；红冲净额变负；自动拆零处方退药回补到整盒行致库存虚增；效期 date-only 口径（FEFO 与过期锁定自相矛盾）
  - **断链**：GORM 零值更新致「药品无法停用」「诊疗项目编码必被毁」；`PUT /users/:id` 省略 role 致账号锁死；状态流转抹除审核痕迹
  - **架构收敛**：账务口径统一为唯一凭证（000040）；业务单号号段表支持 5~20 副本（000041）；JWT 令牌可吊销（000039）；软删与全局唯一约束冲突 + 17 表补 `deleted_at` 索引（000038）
  - **连通性**：前后端 6 处字段契约断裂（库存/采购/收货/拆零/药学服务共 8 列全空白、库存搜索静默失效）；麻精开方 UI 打通 + 限量绕过收口
- 🔭 **二期**：医保真实结算对接（`port` 接口已预留，暂不实现）

---

## 🔭 二期规划

通过 `internal/service/port` 三接口可插拔接入诊疗模块：

| 能力 | 一期 | 二期扩展 |
|------|------|---------|
| 🧑‍⚕️ 患者 | 完整档案 + 过敏史（已实现） | 就诊/病历/检查档案 |
| 📝 处方开立 | 药房端手工录入 | 医生端开立自动传入 |
| 🗃️ 库存 | 药房内部调用 | 诊疗模块直接调用 |
| 💰 计价 | 药费 + 计费闭环 | 挂号费/诊疗费/药费合并结算 |

详见 [docs/05-二期预留接口设计.md](docs/05-二期预留接口设计.md)；二期就诊/病历/收费模块的完整规划（表结构、API、页面）见 [docs/20-二期就诊模块规划.md](docs/20-二期就诊模块规划.md)。

---

## 📄 文档与变更

| 文档 | 说明 |
|------|------|
| 📋 [CHANGELOG.md](CHANGELOG.md) | 版本与变更记录 |
| 📚 [docs/README.md](docs/README.md) | 开发文档总览（26 篇编号文档） |
| 🔍 [docs/14-现状分析与下一步建议.md](docs/14-现状分析与下一步建议.md) | 全量审阅发现与修复进度 |
| 🩺 [docs/15-诊疗模块复审报告.md](docs/15-诊疗模块复审报告.md) | 诊疗模块业务逻辑/漏洞复审与前端搭建参考 |
| 🌐 [docs/16-前端开发就绪评估与对接指南.md](docs/16-前端开发就绪评估与对接指南.md) | 前端就绪评估、页面-接口对照与对接须知 |
| 🗺️ [docs/17-前端开发指南与进度规划.md](docs/17-前端开发指南与进度规划.md) | 前端技术选型、工程结构、页面规划与 6 阶段进度计划 |
| 👥 [docs/18-跟诊护士与药房护士角色设计.md](docs/18-跟诊护士与药房护士角色设计.md) | 护士角色细化：跟诊护士/药房护士职责、权限矩阵与配套 |
| 🧪 [docs/07-测试方案.md](docs/07-测试方案.md) | 测试方案与覆盖策略 |
| 🚢 [docs/10-部署运维.md](docs/10-部署运维.md) | 部署与运维指南 |
| 🛠️ [docs/19-前端工程化提升方案.md](docs/19-前端工程化提升方案.md) | 前端 5 项提升建议评估与融合 |
| 🏥 [docs/20-二期就诊模块规划.md](docs/20-二期就诊模块规划.md) | 二期就诊/病历/收费模块详细规划（S1 表结构 → S7 前端） |
| 🧾 [docs/24-健康检查冲刺实施总结.md](docs/24-健康检查冲刺实施总结.md) | 健康检查冲刺六阶段实施总结与验证结果 |
| 🛡️ [docs/25-第三轮审计修复报告.md](docs/25-第三轮审计修复报告.md) | 第三轮审计复核：回归修复、迁移增量执行、遗留项收尾与全量实测证据 |
| 🛡️ [docs/26-第四轮审计修复报告.md](docs/26-第四轮审计修复报告.md) | 第四轮生产差异收口：处方 FK、网关限速/安全头、ECharts 按需、分层覆盖率 |
| 🛡️ [docs/27-第五轮业务完善报告.md](docs/27-第五轮业务完善报告.md) | 第五轮业务完善：S7 报表/CSV 导出/库房可配置/日志归档/站内通知/打印/测试守护 |
| 🛡️ [docs/28-第六轮审计修复报告.md](docs/28-第六轮审计修复报告.md) | 第六轮：Docker 构建双重阻断、收货质检门禁、**账务口径统一为唯一凭证**、**单号号段表支持 5~20 副本**、JWT 令牌吊销、效期 date-only 口径、前后端字段契约 |

---

<div align="center">

**💊 药房管理系统** — 让药房进销存与处方调配更专业、更安全

⭐ 如果这个项目对你有帮助，欢迎 Star 支持！

</div>
