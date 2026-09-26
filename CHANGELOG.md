# 更新日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 与 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### 修复（第三轮：审计收尾与运维加固）

> 复核 v1.4.0 修复质量后补齐的遗留项；全部经 `go build/vet/test`、`golangci-lint`、
> `go test -tags=integration`（真实 PostgreSQL 16）与 HTTP 冒烟（18 项）实测。
> 完整报告见 [docs/25-第三轮审计修复报告.md](docs/25-第三轮审计修复报告.md)。

- **迁移可重复执行**：新增 `scripts/migrate.sh`（`schema_migrations` 版本表 + 增量执行 + 单文件单事务），
  替换 compose / `deploy/Dockerfile.migrate` / Makefile / CI 中「全量重放 `*.up.sql`」的旧实现——
  旧实现在已初始化的库上二次执行必然失败（`CREATE TABLE` 已存在），生产二次部署与升级会被阻断。
  历史库首次升级用 `YF_MIGRATE_BASELINE_TO=000032` 一次性建立基线（未指定时明确报错而非猜测）。
- **默认口令启动门禁**：release 模式若仍有启用账号使用默认口令 `admin123`，服务拒绝启动
  （`internal/server/startup.go`，可用 `YF_AUTH_ALLOW_DEFAULT_PASSWORDS=true` 临时豁免，仅限演示）；
  debug 模式打印告警并列出账号。
- **写操作审计全覆盖**：新增 `middleware.AuditWrites`，所有已认证的非只读请求落 `operation_logs`
  （动作名细化为 `dispense`/`confirm-dispense`/`return`/`void`/`complete` 等子资源动作，
  记录路由模板、`resource_id`、IP 与状态码；不含请求体，避免口令入库）。此前仅登录/改密/建用户有审计。
- **预警去重（L6）**：新增迁移 `000033` 的 open 状态部分唯一索引 + 仓储 `CreateIfAbsent`
  （`ON CONFLICT DO NOTHING`）；同时修复「低于下限」预警查询键 `'ALL'` 与写入键空串错配导致的
  每次调度重复新增预警。
- **患者档案 mass-assignment（L2）**：写接口改为 `patientWriteRequest` 白名单 DTO，
  仓储更新改为显式列 + map 形式（仍支持布尔/文本清零），客户端无法改写 `id`/`created_at`。
- **空安瓿核对人落库**：`AmpouleReturnRepo.Verify` 一并写入 `verified_by` 并保证 pending→verified 幂等；
  此前仅改状态、核对人不落库（五专审计字段丢失）。
- **事务内一致读**：`InteractionService.CheckPrescription` 与 `availablePacksTx` 支持传入 `tx`，
  Review 持处方行锁期间不再于另一连接查询（缩短锁持有、读一致）。
- **死锁与退药顺序**：调拨按 `inventory_id` 升序加锁（反向并发调拨不再交叉等待）；
  退药记录查询改为「拆零优先」（散片退药不再因先命中整盒记录而报 3013）。
- **金额工具**：`ItemAmount` 对 `packSize<=0/==1` 显式处理（除零防护）；
  `FromYuan(float64)` 改为 `ParseYuan(string)` 精确十进制解析；拆零取整残差（24×42=1008 分）单测固化。
- **CORS 语义澄清**：注释与实现对齐为 fail-closed（白名单为空 = 不下发任何 CORS 头），并补单测锁定。
- **测试补齐与隔离**：新增并发入库（同批次 upsert）、采购超收双护栏、盘点差异口径三项集成测试；
  `setupTestDB` 清理表补 `stocktakes/stocktake_items/split_orders/requisition_*` 等，
  修复盘点残留导致后续用例随机失败（测试顺序相关）；新增 DryRun SQL 断言测试（无需数据库即可守护
  upsert/条件更新/差异口径不被回退）。
- **其他**：`make env-init` 生成含随机密钥的 `.env`（修复 compose 去除弱默认值后 `make db-up` 失败）、
  根 `.env.example`；前端 Nginx 不再反代 `/swagger/`（Swagger 仅非 release 注册）；
  CI 集成测试自带测试密钥（不依赖本地配置）；本地 `configs/config.yaml` 与 `.env` 口令对齐。

### 修复（第四轮：生产差异收口）

> 承接第三轮「遗留与建议」与本轮生产审查；完整报告见
> [docs/26-第四轮审计修复报告.md](docs/26-第四轮审计修复报告.md)。

- **处方 patient_id 外键**：`Prescription.PatientID` 改为 `*int64`，迁移 `000034` 将 0/失效引用规范为 NULL，
  并加 `fk_prescriptions_patient`（ON DELETE SET NULL，处方不随患者档案级联删除）。
- **审计记录登录名**：写操作审计、改密、新建用户改记唯一登录名（不再用可能重名的姓名）；
  登录失败审计不再把非业务错误的底层信息写入 `operation_logs`。
- **网关登录限速 + 安全头**：前端 Nginx 对 `/api/v1/auth/login` 按客户端 IP 限速（10 次/分钟 + burst 5）；
  下发 nosniff/DENY/CSP 等安全头并关闭 `server_tokens`；后端补 `SecureHeaders` 覆盖直连 API。
- **ECharts 按需引入**：统一入口只注册柱状/饼图与实际用到的组件，减小报表 vendor chunk。
- **覆盖率分层门槛 + HTTP 层测试**：`scripts/check_coverage.sh` 卡 domain/service/repository/handler/middleware/pkg；
  新增 handler 集成测试（鉴权/RBAC/审计/分页/Swagger+CORS/安全头）；CI Go 版本对齐 1.26.6。
- **其它**：CORS 配置注释与 fail-closed 实现对齐；K8s ConfigMap 默认信任代理网段；
  集成测试库端口可用 `YF_TEST_DB_PORT` 覆盖。

## [v1.4.0] - 2026-09-12

> 首个正式发布基线：全栈交付（Go 后端 + Vue3 前端 + Docker 一键部署），累积两轮安全审计修复与并线合并。

### 修复（第二轮审计：安全 / 并发 / 部署 / 文档）

> 依据第三方工具审计报告（1×CRITICAL、4×HIGH、12×MEDIUM），逐项复核并修复；关键项已自测。

- **部署阻断（C1）**：`start.sh` 原用 `head -c 48 /dev/urandom | tr -dc` 过滤随机字节，
  过滤后平均仅约 11 字符、必然 <32，导致后端因弱密钥校验 crash-loop——Linux 一键部署整条链路不可用。
  改为 `openssl rand -base64 48`（无 openssl 时 `head -c 48 /dev/urandom | base64`）并剥离 CR；
  `start.bat` 改用 `RandomNumberGenerator`（CSPRNG）；两脚本统一生成 ≥32 位 JWT 密钥。
- **采购并发超收（H1）**：完成收货原为「无锁读校验 + 盲累加」，多张收货单并发完成会超订购量入库。
  新增 `UpdateReceivedWithinLimit` 条件更新（`WHERE received_quantity + ? <= quantity` 原子判定），
  收货创建时按 `SumPendingByOrderItem` 扣除在途（待质检）量，杜绝超收与重复占用。
- **弱 JWT 密钥黑名单（H2）**：扩展 `isWeakJWTSecret` 覆盖连字符/下划线变体与项目默认值
  （`change-me`/`change_me`/`changeme`/`placeholder`/`yaofang-*-secret` 等），
  `docker-compose.yml` 的 `YF_AUTH_JWT_SECRET` 改必填（去掉 `yaofang-compose-dev-secret-change-me` 默认）。
- **默认口令（H3）**：`docker-compose.yml`/`docker-compose.prod.yml` 的数据库口令与 JWT 均改 `:?` 必填，
  `start.sh`/`start.bat` 随机生成数据库口令写入 `.env`；`compose.logging.yml` 的 Grafana 口令改必填。
- **供应链漏洞（H4）**：Go 工具链 `1.26.5 → 1.26.6`、`quic-go v0.59.0 → v0.59.1`，修复 govulncheck 可达漏洞。
- **并发入库回退失效（M1）**：改用 `ON CONFLICT` 原子 upsert（依赖 `uq_inventory`），
  避免 PostgreSQL 唯一键冲突后事务 aborted（25P02）导致并发入库整单失败。
- **盘点差异写错（M2）**：`difference` 由误写实盘数改为 `counted_quantity - book_quantity`。
- **降权不即时（M3）**：鉴权中间件每请求复查数据库当前角色（`UserStateChecker` 返回 role），
  Token TTL 内降权/改角色立即生效。
- **限速可被绕过（M4）**：新增 `server.trusted_proxies`（`YF_SERVER_TRUSTED_PROXIES`）并调用
  `SetTrustedProxies`，默认不信任任何代理头，杜绝伪造 `X-Forwarded-For` 绕过登录限速。
- **CORS/Swagger/健康检查**：release 模式对 `*` 放行告警并支持 `YF_SERVER_CORS_ALLOW_ORIGINS`；
  Swagger 仅非 release 开放；prod 后端健康检查由 `/healthz` 改 `/readyz`（DB 就绪）；
  `readyz` 不再回显数据库底层错误。
- **日志栈（M10）**：Grafana 密码必填、Loki/Grafana 端口仅绑定 `127.0.0.1`、新增数据卷持久化。
- **文档修正（M11）**：README 参考数据 6,794→5,794；CHANGELOG 迁移 33→32；
  docs/01 Go 1.22→1.26；docs/README 更新为全栈交付；docs/02 表清单补全至实际 52 张。
- **前端质量**：`.env.*` 忽略并新增 `.env.example`；`axios` 升级；接入 `vitest` + 14 项单测并入 CI；
  `internal/pkg/auth` 补 JWT 单测。

### 新增（Docker 一键部署包）

- **全容器化交付**：`Dockerfile`（后端多阶段构建，GOPROXY 国内加速）+ `pharmacy-web/Dockerfile`
  （vite build → nginx 托管，SPA 回退 + `/api` 同源反代，浏览器无需 CORS）+ `docker-compose.prod.yml`
  （db 独立卷 `pgdata_prod` 且端口不外发 → 自动迁移 → 后端 release 模式（配置全走 `YF_` 环境变量）→ 前端对外 80）。
- **start.bat / start.sh 一键脚本**：首次运行自动生成 `.env`（随机 48 位 JWT 密钥，`.gitignore` 已排除），
  健康检查等待就绪后输出访问地址；内网其它设备直接浏览器访问 `http://<服务器IP>`。
- **实测**：镜像构建、32 版迁移与种子自动执行（8 账号/ICD-10 1586 条/系统设置）、
  经 nginx 登录与内网 IP 访问全通。

### 新增（前端生产级补全）

- **库存页**：盘点管理全流程 UI（新建冻结→开始→录实盘→差异调整→归档，后端 `GetStocktake` 聚合药名）、
  拆零操作单列表、按片拆零（双人复核）、效期预警数据源；库房下拉替代手填 ID。
- **采购页**：收货单 Tab（列表/详情/质检确认入库）；修复收货效期日期格式（原 `YYYY-MM-DD` 导致后端绑定 400）。
- **交互规则页**（新路由/菜单）：药品配伍禁忌、成分/分类/标签交互规则 CRUD + 药品成分维护。
- **个人中心**（新页面）：资料展示 + 修改密码（强度校验），头部菜单入口；修复侧栏非法 CSS 颜色值。
- **主数据编辑补全**：药品供货关系绑定/解绑、供应商/患者/诊疗项目/药学服务编辑删除、麻精发药专册补录、
  系统管理新增库房管理 Tab 与用户表单校验（含密码强度）。
- **工作台重构**：统计卡（待审核/调配中/就诊中/库存预警/近效期）+ 按权限过滤快捷入口。
- **PatientPicker 通用组件**：就诊/收费/计费筛选与挂号改为患者远程搜索，替换手填 ID。

### 修复（安全与正确性加固，端到端实测）

- **资金安全**：红冲条件更新（`voided=FALSE`）防并发重复冲正；一键计费幂等检查移入事务并锁处方行；
  `CalculateBill` 部分退药按 `ReturnedQuantity` 净额回算、历史费用限定就诊时间窗口；
  收费实收必须等于应收（6106）；结算单按就诊幂等（6107）+ 迁移 `000031` 唯一索引兜底。
- **库存正确性**：领用出库改 `DeductAvailable`（排除预占），处方预占不再被吞；退药回补修复
  LDU 与行口径混用（整盒发药须整盒退回，新增 3013）。
- **安全**：登录限速（IP+用户名 5 次/分，429）；先验密码再查停用防账号枚举（9009）；
  Auth 中间件每请求复查用户状态（停用即时失效）；JWT 弱密钥/默认值拒绝启动。
- **合规**：麻精退药按发药记录行口径写专账负冲正；未质检项禁止入库（2014）。
- **迁移 `000032`**：移除 `special_drug_ledgers`/`ampoule_returns` 对 `prescriptions` 的弱关联外键
  （手工补录 prescription_id=0 此前必然外键违规 500）。
- **测试**：新增 5 个计费/领用集成回归测试；swag 重生成（二期 15 路由入档）；前端生成类型同步。

### 新增（部署运维完善，docs/10）

- **容器化部署**：新增后端 `Dockerfile`（多阶段静态编译 + Alpine 非 root）、前端 `pharmacy-web/Dockerfile`（Vite → Nginx SPA + `/api` 反代）、迁移镜像 `deploy/Dockerfile.migrate`；`docker-compose.yml` 升级为全栈（db → migrate → api → web，依赖健康等待 + healthcheck）；`deploy/k8s/` 提供 Deployment/Service/HPA/迁移 Job/Ingress 全套 manifest。
- **健康检查端点**：`/healthz`（存活，进入优雅关闭后返回 503 摘流量）、`/readyz`（就绪，2s 超时探测数据库连通性）、`/version`（ldflags 注入 version/commit/build_time）；探针请求不写访问日志。
- **结构化日志**：新增 `log.level/format/file` 配置（text/json，生产建议 json），slog 统一 `service=yaofang` + `request_id`；新增 Loki + Promtail + Grafana 聚合栈（`compose.logging.yml` + `deploy/logging/promtail-config.yml`），文档给出裸机 journald/文件采集与 logrotate 兜底方案。
- **数据库连接池说明**：新增 `sslmode`、`conn_max_idle_time` 配置与非法值钳制（idle ≤ open 强制成立）；docs/10 §9 给出容量公式、监控 SQL 与调优建议。
- **CI/CD 流水线**：`ci.yml` Go 版本对齐 go.mod（1.26.5）并新增编译产物步骤；新增 `docker-image.yml`（三镜像推送 GHCR，多架构）与 `deploy.yml`（裸机 systemd 部署：交叉编译 → SCP → 迁移 → 重启 → `/readyz` 探活）；新增裸机部署单元 `deploy/yaofang.service`。
- **无 Docker 可用**：本机无 Docker 不影响以上落地——裸机部署路径（§7）完整可用，Docker/K8s 资产待有环境直接构建。

### 安全加固与健康检查（feature/healthcheck-sprint-2026-09-06）

- **弱 JWT 密钥拒绝启动（Fix/P1）**：`auth.jwt_secret` 为空/`change-me`/`CHANGE_ME` 系列占位值或长度 <32 时 `config.Load` 拒绝启动（与主线 `59c3c71` 的启动校验合并，统一收敛到配置层）；开发模式使用占位值记 `slog.Warn`。杜绝默认密钥静默上线。
- **登录失败审计日志（Fix/P1）**：登录失败写入 `operation_logs`（`action=login_failed`，含失败原因与客户端 IP），与主线登录限速（5 次/分）互补——限速拦截、日志追溯。
- **RequireRoles 健壮性（Fix）**：上下文角色改安全类型断言——未挂载 Auth 时返回 403 而非 panic。
- **依赖升级（Chore）**：`golang.org/x/crypto` v0.54.0→v0.56.0（bcrypt 所在安全库）；连带 `x/mod`/`x/net`/`x/text`/`x/tools` patch 升级。
- **测试补齐（Test）**：`internal/config`（原零测试）新增 6 测试——弱密钥判定/启动校验/连接池钳制/DSN sslmode；`internal/pkg/sanitize` 新增脱敏边界用例；`internal/middleware` 新增探针判定/探针免日志/无 Auth 上下文 3 测试。
- **文档修正（Docs）**：README 三处与 docs/00 的迁移版本计数修正为实际值 32 版（000001-000032）；CHANGELOG v1.0.0 迁移标题 6→12 版本；docs/00 Go 版本 1.22→1.26 对齐 go.mod。
- **与主线并线说明**：rebase 至含 `59c3c71`（登录限速/停用即时失效/资金安全加固）与 `e14f6ab`（Docker 一键部署）的主线之上；Dockerfile/nginx.conf 取两方案并集（版本注入 + 非 root + 国内加速 + 模板化反代），`docker-compose.prod.yml` 注入 `API_UPSTREAM` 适配模板。


### 修复（联调实测）

- **修复 GORM 列名错位导致 INSERT 42703（关键 Bug）**：Go 字段 `VPBBatch` 经 GORM 命名策略
  会转成 `vpb_batch`（V-P-B 错位），而数据库列名为 `vbp_batch`，导致 `drugs`/`vbp_drug_catalog`
  的 `Create` 报「字段不存在」；`internal/model/drug.go` 与 `internal/model/reference.go`
  已显式 `gorm:"column:vbp_batch"` 规避，并新增全模型列名自动审计（schema vs information_schema）确认 0 错位。
- **迁移幂等性补强**：`000004` 索引与 `000014` 软删列改为 `IF NOT EXISTS`，修复在已含 `000001` 更新
  的库上重跑迁移失败的问题。
- **本地全链路联调实测通过**：PostgreSQL 16.4 全新迁移（29 版）+ `scripts/smoke_test.py`
  18 步 HTTP 冒烟全通（登录→药品→采购→入库→调拨→拆零→处方→审核→调配→发药→退药→报表）+
  `go test -tags=integration` 全绿。

### 新增（前端报表增强）

- **ECharts 报表图表**：`pharmacy-web` 报表页接入 ECharts——进销存汇总柱状图、效期分析饼图
  （新增 `ChartPanel.vue` 通用图表组件），已通过 lint/format/type-check/build 全部门禁。
- **二期就诊模块规划**：新增 [docs/20-二期就诊模块规划.md](docs/20-二期就诊模块规划.md)，
  明确就诊/病历/收费模块 S1 表结构、S2-S7 接口与前端页面规划。

### 新增（二期就诊模块后端，docs/20 S1-S5）

- **S1 迁移**（`000029`）：`visits`（挂号/分诊，状态机 waiting→visiting→finished/cancelled）、
  `medical_records`（一就诊一病历：主诉/现病史/体征）+ `medical_record_diagnoses`（多诊断 ICD-10）、
  `charges`（合并结算单：合计/优惠/应收/实收，状态机 pending→paid→refunded）+ `charge_items`（多费用项明细），
  `prescriptions` 增加 `visit_id` 关联就诊。
- **S2 就诊域**：挂号/分诊（自动生成就诊号）、列表（患者/医生/状态/日期筛选）、接诊/结束/退号。
- **S3 病历域**：保存/读取就诊病历（含结构化多诊断，全删全插）。
- **S4 结算域**：`port.IPricingService.CalculateBill` 按就诊聚合药费（已发药处方快照）+
  诊疗项目/耗材计费记录（未红冲）；生成结算单、收费、退费。
- **S5 处方联动**：处方 `visit_id` 关联就诊并校验患者一致；`source` 扩展
  `outpatient/inpatient/refill`（`PrescriptionSourceManual` 保持默认）。
- **实测**：`scripts/smoke_test_phase2.py` 24 步冒烟全通（挂号→接诊→病历→开方→发药→合并结算→收费→退费）。

### 新增（二期就诊模块前端，docs/20 S6）

- **就诊工作台**（`/visits`）：就诊列表（状态/患者/日期筛选）、挂号/分诊、接诊/结束/退号、病历抽屉
  （主诉/现病史/体征/结构化多诊断保存）、一键生成合并结算单。
- **收费台**（`/charges`）：结算单列表与详情（费用明细）、收费（实收金额）、退费。
- 新增 `api/clinical2.ts` 对接就诊/病历/结算接口；`VISIT_STATUS`/`CHARGE_STATUS` 常量入 business.ts；
  菜单与路由注册（就诊工作台=patient:read，收费台=billing:view）。

### 新增（管理员能力）

- **自定义默认诊费**（迁移 `000030`）：`system_settings` 表 + `GET/PUT /system-settings`（UserAdmin），
  键 `default_registration_fee`/`default_consultation_fee`（分，非负整数校验）；
  `CalculateBill` 合并结算时自动带入默认挂号费/诊查费（值>0 且就诊无同类费用行时），
  契约测试 `TestCalculateBillContract` 固化。
- **操作日志审计增强**：`/operation-logs` 增加 `start`/`end` 时间窗口筛选（仓库/服务/Handler 贯通）；
  管理员页日志 Tab 增加用户ID/动作/日期筛选与角色/IP/详情列。
- **账号密码管理**：管理员页「编辑用户」支持重置密码（留空不修改，后端 `UpdateUser` 已支持）；
  新建用户覆盖全部 8 类角色。
- 管理员页新增「系统设置」Tab（挂号费/诊查费配置）；`scripts/smoke_test_admin.py` 冒烟全通
  （诊费配置+非法值拒绝+新建用户+重置密码+新密码登录+日志三类筛选）。

### 修复（针对 docs/14 审阅发现的问题）

- **RBAC 角色权限落地**：所有业务模块写操作按角色矩阵分组——处方开立（ClinicalStaff）、
  处方执行/库存写/药学服务/计费（PharmacyStaff）、药品/分类/配伍/交互规则/特殊药品目录（DrugAdmin）、
  采购/供应商（PurchaseStaff）、报表（ReportAccess）、用户管理（UserAdmin，含药房主任）。
  此前除用户管理与处方审核外，所有路由实际对任意登录用户开放（护士可开方、采购员可改库存等越权）。
- **修复「退回医生」死路**：`Review(action=return)` 由「保留预占」改为**释放预占**（保持 `pending_review`），
  与 `Update` 的「无预占才可改」不变量一致，医生修改后可重新提交再预占（此前退回后因存在预占而无法修改）。
- **麻精「双人核对」强制**：`confirm-dispense` 的核对人改为当前登录用户（不再由客户端指定 `checker_id`）；
  特殊药品（麻醉/精神）发药强制「调配人 ≠ 核对人」双人分离（`4003`）且调配/核对须为药师角色。
- **v1.3 参考数据接线**：ICD-10/集采/医保/耗材/非医保 5 张参考表新增只读查询 API
  （`GET /reference/*`，keyword/拼音码模糊搜索 + 分页），此前数据已入库但零代码接入。
- **工程化清理**：错误码内联魔法数集中到 `pkg/errs`（1006-1009、2009-2012、4003）；
  审核角色判断改枚举；Swagger 版本统一为 1.3.0；全库 `gofmt` 对齐。

### 新增（P1 功能补强）

- **患者档案落地**（迁移 `000020`）：`patients` + `patient_allergies` 表，`prescriptions` 增加 `is_lactating`；
  完整实现 `port.IPatientService`（替换 `SimplePatientService`），新增患者档案/过敏史 CRUD 接口
  （`/patients`、`/patients/:id/allergies`）。处方录入支持 `patient_id` 与 `is_lactating`，
  审核时自动带入过敏史（3010）与哺乳期慎用（3011）检查——激活交互引擎此前「能力就绪但无数据入口」的患者级检查。
- **预警处置闭环**（迁移 `000021`）：`stock_alerts` 记录处理人与处理时间，
  `POST /inventory/alerts/:id/resolve` 支持 `resolved`/`ignored` 状态流转（操作人取自 JWT）。
- **药品↔医保/集采目录匹配标注**（迁移 `000022`）：`drugs` 增加 `insurance_class`/`vbp_batch` 字段，
  新增 `GET /reference/drug-match?name=` 按药品名精确匹配医保类别与集采批次，供录入端自动标注。
- **计费闭环**（迁移 `000023`）：`charge_records` 增加来源单据字段（`ref_type`/`ref_id`），
  新增 `POST /charge-records/from-prescription/:id` 从已发药处方按发药快照价生成计费（幂等）；
  退药时联动写入负金额冲正记录（按处方快照价，混合口径与开方计价一致）。
- **拆零统计报表**（docs/13 F5）：`GET /reports/split-statistics` 按药品聚合期间拆零盒数/入片数/损耗片数，
  含拆零成本/收入/毛利（成本来自拆零操作单、收入来自拆零发药记录）。
- **拆零操作单**（docs/13 F4，迁移 `000024`）：`split_orders` 表记录操作人/复核人/原因/来源行/结果行，
  拆零接口支持 `reviewer_id` 复核人；麻精药品拆零强制双人复核（`2013`，复核人 ≠ 操作人）。
  新增 `GET /inventory/split-orders` 列表与详情。
- **处方剂量一致性校验**：明细 `single_dose`/`total_daily_dose`/`days` 非负校验；
  数量不超过「日总剂量 × 天数」上限（`3012`），防录入错误。

### 工程化（P2 收尾）

- **CI 接入 golangci-lint**（`.golangci.yml`：govet/errcheck/staticcheck/ineffassign/unused/gosimple/gocyclo/misspell），
  本轮清零全部问题（含 1 处领用出库流水未检错误、1 处无效赋值、2 处死代码）。
- **CI 覆盖率门槛**：`scripts/check_domain_coverage.sh` 校验 domain 包 ≥ 85%
  （补齐 enum/prescription/rule 测试至 100%、interaction 至 92.7%）。
- **本地集成测试环境**：`docker-compose.yml`（postgres:16 + 自动迁移）+ `make db-up/db-down`。
- **补测试**：中间件 RBAC 403 用例、`itemAmount` 计价口径用例、过敏匹配/去重/状态机/角色分组用例。

### 诊疗模块复审修复（docs/15，迁移 `000025`）

- **H1 布尔字段持久化**：处方/患者更新改 `Select("*")` 全量更新，`is_pregnant`/`is_lactating` 由 true 改 false 可正常落库（此前 GORM 结构体更新跳过零值，孕期标记无法取消）。
- **H2 计价契约**：`CalculatePrescriptionAmount` 的 `PriceLine.RefID` 改为 `drug_id`（此前误用明细 ID）。
- **H3 退药冲正原子化**：冲正并入 `Return` 事务（`RefundPrescriptionTx`），任一失败整体回滚，账实一致。
- **M1 错误映射**：患者 404/重复卡号 409（`6001/6002`）、诊疗项目重复编码 409（`6003`），替代原始 DB 错误 500。
- **M2 处方列表筛选**：`GET /prescriptions` 新增 `patient_id`/`prescription_type` 参数（repo 早已支持）。
- **M3 权限收窄**：计费查看收窄到「药房人员 ∪ 报表权限」（Billing 组）；患者档案查看收窄到 Pharmacy。
- **M4 计费校验**：负数量/负单价拒绝、`item_type` 枚举校验、项目存在且启用校验；0 元免费项允许。
- **M5 患者校验**：过敏 severity 限 1-3；患者更新零值问题随 H1 一并修复。
- **M6 处方详情聚合**：`GET /prescriptions/{id}` 返回关联患者档案与过敏史（`patient`/`allergies`）。
- **G2 结构化诊断**：`prescriptions.diagnosis_code`（ICD-10）落库 + 编码存在性校验（`6006`），开方输入支持。
- **G3 患者信息回填**：开方关联 `patient_id` 时自动回填姓名/性别/年龄/卡号，患者哺乳标记自动带入。
- **G4 计费筛选**：`GET /charge-records` 支持 `ref_type/ref_id/item_type/patient_id/start/end` 筛选。
- **G5 计费红冲**：`POST /charge-records/:id/void` 标记原单红冲 + 写负金额冲正单（幂等，`6005`）。
- **G6 患者费用视图**：`GET /reports/patient-charges` 按患者聚合收费/冲正/净额（红冲单不计入）。
- **G7 诊疗项目启停用**：`PATCH /clinical-services/:id/status`。
- **L1/L3/L5**：冲正记录以 `Amount` 为准（UnitPrice=0）；未发药处方计费拦截（`6004`）；`port.Patient` 透出 `IsLactating`。
- **计费/患者隐私**：`charge_records` 与 `patients` 增加 `patient_id` 关联字段。

### 前端就绪（docs/16）

- **CORS 跨域支持**：新增 `middleware.CORS` + `server.cors_allow_origins` 配置
  （开发默认放行 `*`，生产限定域名；鉴权走 Authorization 头，无需 credentials）。
- **前端对接指南**：`docs/16-前端开发就绪评估与对接指南.md`（就绪评估矩阵、角色菜单建议、
  页面-接口对照、前端须知：`patient_id=0` 语义 / 冲正负金额口径 / 分金额 / 时间格式 / 类型生成）。
- **前端开发指南与进度规划**：`docs/17-前端开发指南与进度规划.md`（依据后端模块/路由矩阵的技术选型
  Vite+Vue3+TS+Element Plus、工程结构、基础设施设计、12 域页面规划、通用组件清单、
  6 阶段进度计划（约 26 人日）与 M0-M6 验收标准、质量门禁与风险应对）。

### 角色细化：跟诊护士与药房护士（docs/18，迁移 `000026`）

- **角色拆分**：原单一 `nurse` 细化为 **`clinic_nurse`（跟诊护士）** 与 **`pharmacy_nurse`（药房护士）**，
  8 角色体系。迁移 `000026`：`nurse` → `pharmacy_nurse` 改名 + 新增 `clinic_nurse` 种子用户（密码 admin123）。
- **跟诊护士职责**：患者信息管理（`patients` 写，新增 PatientAdmin 组）、**核对医嘱**
  （新增 `POST /prescriptions/{id}/verify-order`，写审计日志不改状态）、辅助录入医嘱/处方草稿（ClinicalStaff）、
  诊疗执行计费（ChargeStaff）。
- **药房护士职责**：药品/库存查询、**医疗耗材领用/补发登记**（`POST /inventory/requisition`）、
  处方执行辅助（PharmacyStaff）、**辅助核对收费**（Billing 查看）、药学服务。
- **权限分组新增/调整**：`PatientAdmin`/`PatientRead`/`ChargeStaff` 三组；`PharmacyStaff` 以 `pharmacy_nurse`
  替换 `nurse`；`ClinicalStaff` 增加 `clinic_nurse`。
- 患者档案写由 Clinical 改 PatientAdmin、读由 Pharmacy 改 PatientRead；计费写由 Pharmacy 改 ChargeStaff。

### 业务闭环补全（迁移 `000027`）

- **耗材领用/补发登记单**（药房护士核心职能）：`requisition_orders` + `requisition_order_items` 表，
  新增 `POST /inventory/requisition-orders`（多明细 LDU 口径 FEFO 扣减，拆零优先、整盒按 pack_size 折算）、
  列表与详情接口（按库房/目的筛选）。
- **患者用药史**（docs/15 L6）：实现 `GetMedicationHistory`，新增 `GET /patients/{id}/medication-history`。
- **双护士 RBAC 集成用例**：`rbac_integration_test.go`（跟诊护士可建档不可领用，药房护士相反，CI 验证）。

### 开方精细化（迁移 `000028`）+ 前端 P3-P6

- **给药途径与分批组**（迁移 `000028`）：`prescription_items` 新增 `route`
  （oral/external/iv/im/iv_drip/inhale/other，枚举校验）与 `batch_group`（口服组/输液组1 等），
  支撑口服/外用/注射剂开具与分批配伍分组，开方不混淆。
- **前端 P3 开方页重做**：按给药途径 + 分批分组卡片（彩色分组头、组内多明细、途径选择、静滴提示），
  保存时按组落 `route`/`batch_group`；处方详情展示分组/途径。
- **前端 P4**：计费管理（7 维筛选/红冲/手工计费/诊疗项目目录启停用）、药学服务（咨询/不良反应/指导）、
  特殊药品（麻精处方/空安瓿回收/专账）。
- **前端 P5**：报表中心（进销存/效期/特殊/工作量/拆零/患者费用 6 报表）、系统管理（用户/操作日志）。
- **前端 P6**：新增前端 CI（frontend-ci.yml：install + type-check + eslint + prettier + build）；
  全量验证（type-check/lint/format/build）通过。

## [v1.3.0] - 2026-08-02

### 基础参考数据（种子）

- **ICD-10 疾病诊断编码**（迁移 `000015`）：`diagnosis_codes` 表，1,586 条诊断编码，含 22 个一级章节 + 220 个二级分类的层级结构。数据来源 [ICD-10-CN](https://github.com/chaseliu/ICD-10-CN)。用于诊断名称模糊搜索/自动补全。
- **国家集采药品目录**（迁移 `000016`）：`vbp_drug_catalog` 表，392 个品种（去重），覆盖第 1-10 批国家药品集中带量采购全部批次（2018-2025）。含药品通用名、剂型、剂型分类、集采批次。用于药品名称模糊搜索/自动补全，后续入库时再精确填列规格/厂家/价格。
- **国家医保药品目录**（迁移 `000017`）：`nhsa_drug_catalog` 表，**3,313 条**（西药+中成药，按药品名称+剂型去重），含甲类 749 条 + 乙类 2,564 条。数据提取自国家医保局《2024年版国家医保药品目录》官方PDF（200页）。含药品名称、剂型、甲乙类、药品分类、子分类。覆盖药房日常用药的绝大部分品种。
- **医用耗材目录**（迁移 `000018`）：`medical_consumables` 表，**141 类**常用耗材。涵盖注射器具、输液器具、采血器具、导管、敷料、缝合材料、麻醉耗材、手术室耗材、骨科耗材、眼科耗材、消毒用品、护理用品、检验耗材、影像耗材等分类。含NMPA管理类别（Ⅰ/Ⅱ/Ⅲ类/消字号）。
- **非医保常用药品**（迁移 `000019`）：`non_insurance_drugs` 表，**362 种**（已与医保目录去重，已排除罕见病药和抗癌药）。涵盖维生素/矿物质保健品、感冒咳嗽OTC、皮肤科外用药、五官科、消化系统、妇科儿科男科、镇痛、骨骼肌肉、神经系统、戒烟减肥、中成药外用贴膏等品类。区分处方药/OTC/保健品/消杀类。弥补医保目录外药店常见的自费品种缺口。

### 新增表

- `diagnosis_codes` — ICD-10 诊断编码查找表（code, disease_name, chapter_code/name, category_code/name, py_code）
- `vbp_drug_catalog` — 国家集采药品参考目录（generic_name, dosage_form, dosage_category, vbp_batch, py_code）
- `nhsa_drug_catalog` — 国家医保药品目录（drug_name, dosage_form, insurance_class, drug_category, sub_category, notes）
- `medical_consumables` — 医用耗材参考目录（item_name, sub_category, category, nmpa_class, description）
- `non_insurance_drugs` — 非医保常用药品参考目录（drug_name, dosage_form, rx_otc_class, category, sub_category, description）

## [v1.2.0] - 2026-08-01

### 用户角色体系（7 种角色）

- **角色列表**：管理员（admin）、药房主任（pharmacy_director）、药师（pharmacist）、医生（doctor）、护士（nurse）、采购员（buyer）、财务（finance）。
- **权限分组**：PharmacyStaff（药房管理）、ClinicalStaff（处方开立）、ReportAccess（报表查看）、UserAdmin（用户管理）。
- **角色校验**：`CreateUser`/`UpdateUser` 强制校验角色合法性（`IsValidRole`），拒绝非法角色名。
- **种子用户**：7 个预置账户（迁移 `000010`），密码均为 `admin123`。
- **调配/核对合并**：不再设独立调配员/核对员角色，由医生和药师兼任调配与核对职能。
- **护士权限**：拥有处方执行权（调配/发药/核对/药品查询），无处方开立权。

### 药师审核流程

- **强制审核**：`POST /prescriptions/:id/review` 仅 pharmacist / pharmacy_director / admin 可执行。
- **退回医生**：新增 `action: "return"`，药师审核发现问题时退回医生修改——保留预占库存、处方保持 `pending_review`，医生可直接修改后重新提交。
- **三种审核动作**：`pass`（通过→调配）、`reject`（驳回→释放预占）、`return`（退回→医生修改）。

### 药房物品类型与临床边界

- **`item_type` 字段**（迁移 `000011`）：`drugs` 表区分药品（drug）与耗材（consumable），统一走进销存。
- **临床诊疗项目**（迁移 `000012`）：`clinical_services` 表管理手法复位、静脉注射等不入药房库存的诊疗项目，独立计价。
- **计费记录**：`charge_records` 表统一药品/耗材/诊疗项目计费入口，护士/医生可直接录入。
- **领用出库**：`POST /inventory/requisition` 医护内部消耗出库（不计费），FEFO 扣减。
- **架构边界**：药房物品（drugs 表）vs 临床诊疗项目（clinical_services 表）清晰分离，避免二期扩展时混淆。

### 代码审查与修复

- 两轮对抗性代码审查共发现 32 个问题，修复 14 个（含 3 CRITICAL + 5 HIGH）。

## [v1.1.0] - 2026-08-01

### 新增：药物相互作用引擎

- **多策略分层匹配引擎**（`internal/domain/interaction/`）：支持显式药品对、成分级、分类级、标签级 4 层匹配，自动去重（explicit > ingredient > class > tag），按严重程度分级（禁忌=拦截 / 慎用=警告 / 注意=提示）。
- **患者个体化禁忌检查**：年龄禁忌、妊娠分级禁忌（A/B/C/D/X）、哺乳期慎用、过敏史交叉匹配。
- **重复用药检测增强**：同通用名 + 同活性成分 + 同药理分组三重检测。
- **标签系统**：预定义 15 个核心交互标签（nsaid、anticoagulant、maoi、ssri、qt-prolonging、cyp3a4-inhibitor 等），支持标签间交互规则。
- **37 条种子交互规则**：12 成分级 + 11 分类级 + 8 标签级 + 6 患者禁忌（迁移 `000008`）。

### 新增：数据库迁移 000007-000008

- `000007`：`drugs` 表新增 9 个临床字段（active_ingredient、atc_code、pharmacological_group、pregnancy_category、age_min/max_years、interaction_tags、lactation_safe、contraindication_notes）；`drug_interactions` 表新增 4 个循证字段（mechanism、evidence_level、source_reference、updated_at）；新建 6 张表（drug_ingredients、ingredient_interactions、class_interaction_rules、tag_interactions、patient_contraindications、interaction_results）。
- `000008`：37 条核心交互规则种子数据。

### 新增：交互规则管理 API（14 个端点）

- `GET/POST/PUT/DELETE /api/v1/ingredient-interactions` — 成分级交互规则 CRUD
- `GET/POST/PUT/DELETE /api/v1/class-interactions` — 分类级交互规则 CRUD
- `GET/POST/PUT/DELETE /api/v1/tag-interactions` — 标签级交互规则 CRUD
- `GET/POST /api/v1/drugs/:id/ingredients` — 药品成分映射
- `DELETE /api/v1/drug-ingredients/:id` — 删除成分映射

### 修复：API 缺口补齐（5 个端点）

- `POST /api/v1/auth/logout` — 登出端点（设计文档已有，代码缺失）
- `PUT /api/v1/medication-guidances/:id` — 用药指导更新
- `DELETE /api/v1/medication-guidances/:id` — 用药指导删除
- `GET /api/v1/special-drugs/prescriptions` — 麻精处方登记列表
- `GET /api/v1/drugs/:id/availability` — 药品可用库存别名路由
- `GET /api/v1/special-drugs/reports/usage` — 特殊药品使用统计别名路由

### 增强

- **处方审核响应结构化**：`POST /prescriptions/:id/review` 返回 `AuditReviewResult`（含 warnings/drug_warnings 明细），审核人可看到具体提醒项而不仅是错误码。
- **ErrDuplicateInteraction(1005) 正式启用**：配伍禁忌重复创建时返回专用错误码。
- **PrescriptionFilter 新增 `prescription_type` 筛选**：支持按处方类型查询。

## [v1.0.0] - 2026-08-01

药房管理系统一期交付。Go + PostgreSQL 模块化单体后端，覆盖药品进销存与处方调配全流程，
预留诊疗模块扩展接口。详细设计见 [docs/](docs/README.md)。

### 一期核心交付

- **药品主数据**：一药多规/一品多商、分类、抗生素分级、特殊管制标记、配伍禁忌、启停用/冻结。
- **供应商与采购**：供应商维护、供货关系、采购单状态机、质检收货（批次/效期绑定）、防超收。
- **库存**：批号效期全程追踪、FEFO 发药、预占/实扣/释放、调拨、盘点、效期与上下限预警、采购计划建议。
- **处方**：录入 → 审核（配伍/极量/重复用药）→ 调配 → 发药 → 退药全状态机，审计日志可追溯。
- **特殊药品「五专」**：麻精专用处方、双人核对、专账联动、空安瓿回收。
- **药学服务**：用药咨询、不良反应登记、用药指导。
- **报表**：进销存汇总、效期分析、特殊药品使用统计、调配工作量。
- **二期预留**：`internal/service/port` 三接口（患者/库存/计价）+ 简易实现，契约测试固化。

### 质量与工程化

- 独立对抗性代码复审：修复 12 个真实缺陷（重复提交超扣、拆零/报损/盘点破坏预占不变量、
  采购超收、质检终态不可达、多明细退药误置终态等），见 [docs/11-代码复审报告.md](docs/11-代码复审报告.md)。
- 单元 / 集成 / 契约测试 + HTTP 冒烟；并发不超卖、幂等提交、退药回补等关键路径全覆盖。
- Makefile 与 GitHub Actions CI（lint + 单元 + PostgreSQL 集成 + Swagger 一致性门禁）。
- 性能专项：补齐 `item_id`/FEFO 索引，热路径 EXPLAIN 验证，见 [docs/12-性能优化报告.md](docs/12-性能优化报告.md)。
- 部署运维文档 [docs/10-部署运维.md](docs/10-部署运维.md)。

### 药品拆零能力（专项完善）

- **拆零成本修正**：拆零行进价取「批次实际进价」折算，不再用主数据价。
- **按片拆零** `POST /api/v1/inventory/split-units`：开盒零头入账 + 破损报损，
  `units + damaged == boxes × pack_size` 账目平齐校验。
- **拆零零售价可配置**：显式配置覆盖公式推算（分摊损耗），进价恒公式推导。
- **混合发药**：处方明细 `quantity` 统一 LDU，精确计价
  `金额 = 整盒数×盒价 + 零头×拆零价`；分配器跨整盒/拆零库存。
- **自动拆零**：零头拆零不足时预占自动预留待拆盒（`need_split`），发药确认自动拆盒入账。
- 完整方案与对照见 [docs/13-药品拆零方案.md](docs/13-药品拆零方案.md)。

### 数据库迁移（12 个版本）

| 版本 | 说明 |
|------|------|
| 000001 | 初始建表（27 张） |
| 000002 | 种子数据（分类/库房/管理员） |
| 000003 | 收货明细关联采购明细（防超收） |
| 000004 | 发药明细/拆零 FEFO 索引 |
| 000005 | 处方明细盒价与可拆零快照（混合发药） |
| 000006 | 预占待拆标记（自动拆零） |
| 000007 | 交互引擎：药品临床字段 + 6 张新表（成分/分类/标签/禁忌/结果） |
| 000008 | 交互规则种子数据（37 条） |
| 000009 | 处方新增 is_pregnant 字段（妊娠禁忌检查） |
| 000010 | 用户角色种子数据（v1.2：7 种角色） |
| 000011 | drugs 表新增 item_type 字段（drug/consumable） |
| 000012 | clinical_services + charge_records 表（诊疗项目+计费） |

### 已知限制（一期有意为之）

- 收货默认入「中心药库」、处方默认从「门诊药房」发药（常量可改）。
- 拆零单（麻精双人复核）、拆零损耗/毛利统计为后续项。
