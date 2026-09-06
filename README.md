<div align="center">

# 💊 药房管理系统

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
| 🏥 二期就诊模块 | 挂号/分诊（就诊状态机 waiting→visiting→finished/cancelled）、病历（主诉/现病史/体征/多诊断 ICD-10）、**合并结算**（药费+诊疗项目费聚合 CalculateBill、收费/退费状态机）、处方关联就诊（source 扩展 outpatient/inpatient/refill） |
| ⚙️ 系统设置 | 管理员自定义**默认诊费**（挂号费/诊查费，合并结算自动带入）、用户账号管理（新建/编辑/**重置密码**）、全量操作日志审计（用户/动作/时间筛选） |
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

### ⚙️ 环境要求

| 依赖 | 版本 |
|------|------|
| Go | ≥ 1.26.5 |
| PostgreSQL | ≥ 14 |
| Docker Compose（可选） | 一键起库 / 全栈部署 |

### 🐳 容器化一键部署（可选，需 Docker）

```bash
docker compose up -d --build    # db → migrate → api → web 全栈
# 前端 http://localhost:8088 ，API 健康检查 http://localhost:8080/readyz
```

无 Docker 的服务器可走 **裸机 systemd 部署**（`deploy/yaofang.service`），K8s 部署见 `deploy/k8s/`，完整说明见 [docs/10-部署运维.md](docs/10-部署运维.md)。

### 🗄️ 1️⃣ 初始化数据库

```bash
# 方式一：Docker 一键起库并执行全部迁移（推荐 🐳）
make db-up                                   # 需 Docker Compose

# 方式二：本机 PostgreSQL（postgres 超级用户）
psql -U postgres -h localhost -c "CREATE ROLE yaofang LOGIN PASSWORD 'yaofang123';"
psql -U postgres -h localhost -c "CREATE DATABASE yaofang OWNER yaofang;"

# 依次执行全部迁移与种子（migrations/NNNNNN_*.up.sql，共 32 个版本）
for f in migrations/*.up.sql; do
  echo "== $f"
  psql -U postgres -h localhost -d yaofang -v ON_ERROR_STOP=1 -f "$f"
done
# 等价：make db-migrate（Linux/Git Bash，可配置 YF_DB_HOST/USER/NAME）
```

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

- 后端已支持 **CORS 跨域**（`server.cors_allow_origins` 白名单，开发默认放行 `*`，生产限定域名）。
- 接口文档：`http://localhost:8080/swagger/index.html`（含全部端点与 TS 类型生成来源 `docs/swagger.json`）。
- 对接约定与前端须知见 [docs/16-前端开发就绪评估与对接指南.md](docs/16-前端开发就绪评估与对接指南.md)。
- 前端技术选型、工程结构、页面规划与进度计划见 [docs/17-前端开发指南与进度规划.md](docs/17-前端开发指南与进度规划.md)。
- 前端工程 `pharmacy-web/`：P0-P6 全部完成（登录/RBAC/主数据/采购库存/患者处方核心流程含给药途径与分批配伍分组/计费/药学服务/特殊药品/报表/系统管理/CI），报表页已接入 **ECharts**（进销存汇总柱状图、效期分析饼图），详见 [pharmacy-web/README.md](pharmacy-web/README.md)。
- **生产级补全**：盘点管理全流程（新建冻结→开始→录实盘→差异调整→归档）、按片拆零（双人复核）、收货单质检确认入库、交互规则可视化维护（配伍/成分/分类/标签 + 药品成分）、供货关系绑定、麻精发药专册补录、个人中心（资料/改密）、库房管理、患者/诊疗项目/药学服务编辑删除；工作台统计卡与按权限快捷入口；患者/药品统一远程搜索选择器。

### ✅ 4️⃣ 测试

```bash
go test ./...                          # 单元测试（domain/service/middleware）
go test -tags=integration ./internal/service/   # 集成测试（需 PostgreSQL；先 make db-up）
golangci-lint run ./...                # 静态检查（CI 门槛）
bash scripts/check_domain_coverage.sh  # domain 覆盖率门槛（≥85%）
python scripts/smoke_test.py           # HTTP 冒烟测试（需服务已启动）
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
│   ├── middleware/        # 🛡️ JWT、日志、恢复、请求ID
│   ├── scheduler/         # ⏰ 定时任务
│   └── pkg/               # 🧰 通用组件（errs/money/pagination/auth）
├── migrations/            # 📦 golang-migrate SQL 迁移（32 个版本）
├── configs/               # ⚙️ 配置样例
├── deploy/                # 🚢 部署资产：systemd 单元、K8s manifests、日志聚合配置
├── docs/                  # 📚 开发文档（20 篇）
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
| 📚 参考数据 | 诊断编码 / 集采 / 医保 / 非医保 / 耗材搜索 · 目录匹配 |
| 📊 报表 | 进销存汇总 · 效期分析 · 特殊药品统计 · 调配工作量 · 拆零统计 · **患者费用聚合** |

---

## 📊 项目状态与路线图

### ✅ 当前状态

| 里程碑 | 状态 |
|--------|------|
| 一期核心闭环（主数据/采购/库存/处方/拆零） | ✅ 已交付 |
| 药物相互作用引擎（4 策略 + 患者级检查） | ✅ 已交付 |
| 参考数据（ICD-10/集采/医保/耗材/非医保，6,794 条） | ✅ 已交付并接线 |
| RBAC 角色矩阵强制 / 麻精双人核对 | ✅ 已落地 |
| 患者档案 + 计费闭环 + 拆零操作单/统计 | ✅ 已交付 |
| 诊疗模块复审修复（docs/15） | ✅ 已交付 |
| 前端就绪（CORS + 对接指南 docs/16 + 开发指南/进度 docs/17） | ✅ 已就绪 |
| CI 门槛（golangci-lint + 覆盖率 ≥85%） | ✅ 已落地 |
| 本地全链路联调实测（PG16 迁移 + HTTP 冒烟 + 集成测试） | ✅ 全绿（PG14-16 兼容） |
| 前端 ECharts 报表增强（进销存汇总/效期分析） | ✅ 已交付 |
| 二期就诊模块 S1-S5（就诊/病历/合并结算后端 + 处方联动） | ✅ 已交付（联调全通） |
| 二期就诊模块 S6（前端就诊工作台 + 收费台） | ✅ 已交付 |
| 管理员能力（默认诊费配置 + 账号/密码管理 + 操作日志筛选） | ✅ 已交付 |
| 安全与正确性加固（登录限速/停用即时失效/重复红冲与重复计费防护/质检闭环/退药单位口径修复） | ✅ 已交付并实测 |
| 前端生产级补全（盘点全流程/收货质检/交互规则维护/个人中心/供货关系/工作台重构） | ✅ 已交付并端到端实测 |
| Docker 一键部署包（Nginx+后端+PG 全容器化，内网浏览器直访，自动迁移/随机密钥） | ✅ 已实测（镜像构建/迁移/登录/内网访问全通） |
| 部署运维完善（容器化 + 健康检查 + 日志聚合 + 连接池 + CI/CD） | ✅ 已交付 |
| 健康检查冲刺（安全加固 + 测试补齐 + 依赖升级 + 文档修正） | ✅ 已交付 |
| 二期就诊模块规划（docs/20） | 📋 S7 报表待实施 |

> 迁移至 `000032`，共 **32 个版本**；质量门禁：`go build` / `go vet` / `go test` / `gofmt` / `golangci-lint` 全绿 ✅

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
- 🔭 **二期**：S7 合并结算报表

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
| 📚 [docs/README.md](docs/README.md) | 开发文档总览（20 篇） |
| 🔍 [docs/14-现状分析与下一步建议.md](docs/14-现状分析与下一步建议.md) | 全量审阅发现与修复进度 |
| 🩺 [docs/15-诊疗模块复审报告.md](docs/15-诊疗模块复审报告.md) | 诊疗模块业务逻辑/漏洞复审与前端搭建参考 |
| 🌐 [docs/16-前端开发就绪评估与对接指南.md](docs/16-前端开发就绪评估与对接指南.md) | 前端就绪评估、页面-接口对照与对接须知 |
| 🗺️ [docs/17-前端开发指南与进度规划.md](docs/17-前端开发指南与进度规划.md) | 前端技术选型、工程结构、页面规划与 6 阶段进度计划 |
| 👥 [docs/18-跟诊护士与药房护士角色设计.md](docs/18-跟诊护士与药房护士角色设计.md) | 护士角色细化：跟诊护士/药房护士职责、权限矩阵与配套 |
| 🧪 [docs/07-测试方案.md](docs/07-测试方案.md) | 测试方案与覆盖策略 |
| 🚢 [docs/10-部署运维.md](docs/10-部署运维.md) | 部署与运维指南 |
| 🛠️ [docs/19-前端工程化提升方案.md](docs/19-前端工程化提升方案.md) | 前端 5 项提升建议评估与融合 |
| 🏥 [docs/20-二期就诊模块规划.md](docs/20-二期就诊模块规划.md) | 二期就诊/病历/收费模块详细规划（S1 表结构 → S7 前端） |

---

<div align="center">

**💊 药房管理系统** — 让药房进销存与处方调配更专业、更安全

⭐ 如果这个项目对你有帮助，欢迎 Star 支持！

</div>
