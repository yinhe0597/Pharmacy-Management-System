# 更新日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 与 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [Unreleased]

### 修复（第六轮：生产阻断收口 + 账务口径统一 + 单号多副本 + 前后端契约守卫）

> 本轮以「能否上线」为标准复审全量代码/配置/部署，**凡可证伪的一律实测**
> （真实 PG16 / Docker build / 依赖库源码 / 并发测试），据此推翻了 3 条静态审计结论。
> 完整报告见 [docs/28-第六轮审计修复报告.md](docs/28-第六轮审计修复报告.md)

**阻断（此前任何容器化路径都必然失败）**

- **`docker build` 双重阻断**：①`Dockerfile:7` 基础镜像 `golang:1.26.5-alpine` 低于 `go.mod` 要求的 `>= 1.26.6`，
  构建在 `go mod download` 即终止；②`.dockerignore:18` 排除 `docs/`，但 `internal/server/server.go:29`
  硬依赖 `yaofang/docs`（swag 生成的路由包）。修复后 `docker build` 实测通过。

**合规**

- **收货质检门禁失效**：`PurchaseReceiptItem.QCResult` 同时带 GORM `default:1` 与 DDL `DEFAULT 1`，
  「未质检(0)」被 GORM 省略该列后由 DB 默认值改写为「合格(1)」，导致 `HasUninspected` 恒 false
  ——收货单可在零质检下入库（违反 GSP）。迁移 `000037` 去掉列默认值（**不回溯改写历史行**：
  无法区分「真合格」与「未登记」）；`Receive` 增加 `qc_result ∈ [0,2]` 取值域校验。
- **麻精限量绕过**：`CheckPrescriptionLimit` 判据为 `Days > limit`，留空（0）即恒假放行——不填天数成为绕限量通路。
  改为 `Days <= 0` 拒绝。
- **麻精限量被宽松类型覆盖**：`deriveSpecialType` 取 `max`，而 `Narcotic=1 < Psycho=2`，
  「麻醉 + 二类精神」被判为二类精神（限 7 日），**麻醉的 3 日限量被跳过**。改为「最严优先」。
- **非法处方类型绕限量**：`prescription_type=99` 落入 `CheckPrescriptionLimit` 的 default 分支跳过校验。
  新增 `IsValidPrescriptionType` 并在入口拒绝越界值。

**正确性**

- **库存调拨负数量致虚增**：`Transfer` 未校验 `Quantity > 0`，负数使 `Available() < qty` 恒真、
  `Deduct` 生成 `quantity + N`，源库房库存凭空增加且 `transfer_out` 流水记为正数（伪入库）。
  service 入口 + `Deduct`/`DeductAvailable` 仓储层双层拒绝。
- **红冲净额变负**：`VoidCharge` 置原单 `voided=true` 并新增 `Amount=-X, Voided=false` 冲正单，
  而下游只按 `voided=FALSE` 过滤 → 只捞到 `-X`，患者净额凭空减少、结算甚至因 `payable<0` 被阻断。
  冲正单同步置 `voided=true`（两单成对，净额归零，留痕由冲正单本身承担）。
- **自动拆零处方退药虚增**：`need_split` 发药记录按拆零形态（片）计量，但 `InventoryID` 仍指向被拆开的整盒行，
  退药回补把「片」写进以「盒」计量的行 → 虚增一个整盒。改为按 `rec.IsSplit` 定位同批次拆零行。
- **效期 date-only 口径统一**（实测确认）：`expiry_date` 是 DATE 列，PG 按会话时区提升为当天 00:00；
  传 `time.Now()`（带时分秒）使 `00:00 >= 14:23` 为 false，当天到期批次被 FEFO **整日排除**，
  而同一参数的 `<` 比较又把它判为已过期并锁定——两条路径自相矛盾，4 条路径给出 3 种结论。
  三个仓储方法改用 `CURRENT_DATE` 并**移除 `today` 形参**（从签名消除陷阱）；
  `todayNow()` 归一化到 Asia/Shanghai 当天零点（容器 TZ=UTC 时原本会差 8 小时、跨零点业务日错一天）。
- **GORM 零值更新两处断链**：`drug_repo` 结构体 `Updates` 跳过零值 → **药品无法停用**、无法关拆零、无法清零零售价；
  `clinical_repo` `Select("*")` 无 `Omit` → PUT 漏传 `code` 写空串，**第一次毁掉唯一编码、第二次必撞唯一键**。
  统一为 `Omit(主键/编码/创建时间).Select("*")` 写法。
- **状态流转抹除审核痕迹**：`UpdateStatus` 无条件写 `reviewed_at`/`auditor_id`/`checker_*`，
  调配阶段未回填即把已记录的审核人与审核时间清空。改为仅在回填时才写入。
- **`PUT /users/:id` 省略 role 致账号锁死**：`role` 是可选字段，无条件赋值导致写入空串 → `RequireRoles` 全线 403 且无 API 可恢复。
- **分次退药终态不可达**：终态判定基于请求局部 map，多明细分次退药时先退的明细在后续请求中查不到。改为以库中快照累加。
- **预警处置静默丢单**：`UpdateStatus` 忽略 `RowsAffected`，已处置的预警重复提交返回成功但未落库。改返回 `(bool, error)`。
- **空安瓿核对人落空**：前端不传 `verified_by` 写入空串，麻精双人核对链在数据层断裂。改取 ctx 登录用户并加非空校验。
- **负折扣 = 任意加价**：`discount` 无非负校验，`payable = total - discount` 可被抬高且必然可实收。
- **自动拆零流水丢操作人**：`addStockTx` 实参错位，`split_in`/`split_out` 流水 operator 为空，麻精双人追溯断链。
- **`StockSettingRepo.Upsert` 并发唯一冲突**：「先 First 后 Create」并发下双双 Create，一方撞 23505 →
  PG 事务 `aborted(25P02)` 整单 500。改 `ON CONFLICT` 原子 upsert。
- **4 处批量写入空切片 500**：GORM 对空切片返回 `ErrEmptySlice`（已读 `gorm@v1.31.2` 源码确认）被当系统异常上报。
- **`Count` 的 error 被丢弃**：DB 抖动时 `n=0` 被误报为「结算单不存在」，掩盖真实故障。
- **交互检查 N+1 + 吞错**：嵌套循环内 `GetByID`，改 `BatchGetDrugProfiles` 批量载入（仓内既有范式）。

**架构收敛**

- **统一记账口径**（迁移 `000040`）：`charges` + `charge_items` 为**唯一记账凭证**；
  `charge_records`（+= `visit_id`）降级为「应收计费项目源」，只被结算单按就诊**精确**归集，
  不被任何报表直接统计。连锁修复：消除按时间窗口猜归属导致的跨就诊错归集；
  放开 `item_type` 排除 `drug` 修掉「手工药品费永远不进账单」的收入漏记；
  排除 `ref_type='prescription'` 防止与处方快照口径**重复计费**；
  `charge_items.source_record_id` 回溯来源，消除 `item_id` 多态引用歧义；
  `PatientCharges` 报表改读同一凭证，与 `RevenueBreakdown` 同源同口径。
  > 保留「仅对 `visit_id IS NULL` 存量行」的时间窗口回退——只认 `visit_id` 会让不传该字段的录入方
  > **静默漏计费**，对药房账目比错归集更危险；同时补上写入路径（手工计费弹窗新增「关联就诊」选择器）。
- **业务单号多副本安全**（迁移 `000041`）：`seq.Next` 原为「前缀+Unix秒+进程内 3 位自增」，只在单进程唯一；
  同秒内两副本各自产生第 N 条即得到相同单号，撞 UNIQUE 约束，而 PG 下唯一冲突会把事务置入 `aborted(25P02)`，
  **整笔业务（含库存扣减）回滚**。改为号段表（`doc_segments`）：单条 `INSERT..ON CONFLICT..RETURNING`
  原子申请 256 个号，进程内独占区间 → **5~20 副本下全局唯一**，DB 往返摊薄到 1/256。
  格式 `<prefix><yyyyMMddHHmmss><8 位绝对序号>`，保留「单号可按时间排序」的现场运维刚需；
  与旧格式长度不同故不与存量单号冲突。`Next` 改为返回 `error`——宁可失败也不产生重复单号。
- **JWT 令牌可吊销**（迁移 `000039`）：新增 `users.token_version` 签入 claims 并逐请求比对；
  改密与管理员重置口令时自增，一次性作废该用户全部存量 token（此前最长 30 天无法处置）。
  附带：`role`/`username`/`name` 改为一律取库中当前值（此前取 token 旧值，改名后操作日志归属失真）。
- **软删与唯一约束的系统性冲突**（迁移 `000038`）：11 张业务主数据表的**全表**唯一约束在软删后仍占用槽位，
  而创建路径预检走默认作用域查不到软删行 → 预检通过 → INSERT 撞 23505 → **接口返回 500 而非 409**。
  改为部分唯一索引 `WHERE deleted_at IS NULL`（沿用 000031/000033 已验证范式）。
  单据流水号（`visit_no`/`charge_no`/`visit_id`）与 `drugs.code` 刻意保持全局唯一。
  附带：17 张软删表**零个** `deleted_at` 首列索引（model 声明了 `gorm:"index"` 但 DDL 只 `ADD COLUMN`），
  每条 List/Count 都隐式 `WHERE deleted_at IS NULL` → 全线退化为顺序扫描，已补齐。

**连通性**

- **前后端 6 处字段契约断裂**（前端类型层给出虚假保证：`vue-tsc` 通过但运行时全空，故长期潜伏）：
  库存列表药品/库房两列全空、采购列表单号/供应商两列全空、收货质检弹窗药品列全空、拆零单入片数列全空、
  不良反应/用药指导药品与反应两列全空；**库存列表「药品名称」搜索完全无效**（前端传 `keyword`，后端无该字段）。
  新增 6 个带 JOIN 的行 DTO 并修正字段名（`units_in`→`units`、`order_no`→`purchase_no`、`reaction`→`reaction_desc`）。
- **麻精开方 UI 打通**：后端早已支持 `prescription_type`，纯前端缺失导致麻精「五专」链路无法从任何 UI 走通。
  开方表单新增处方类型选择器（复用共享字典，不新增第 4 份硬编码副本）、动态合规提示条与提交前拦截。

**安全**

- **审计规避原语**：`AuditWrites` 用请求上下文落库，客户端「读到 200 立刻 RST」即可让敏感写操作
  （发药/红冲/改价/停用）已提交入库而审计行丢失。改 `context.WithoutCancel` + 3s 超时。
- **登录限速 body 无上限**：未认证的登录接口 `io.ReadAll` 无长度限制 → 远程 OOM。加 4 KiB `LimitReader`。
- **限速桶无界增长**：用户名由攻击者完全控制且长度不限，桶只清理「已过期」项 → 内存单调增长。
  用户名截断 64 字节 + 桶数硬上限 4096。
- **登录时序侧信道**：用户不存在时不执行 bcrypt，与密码错误路径相差 60~100ms，可统计区分账号是否存在。
  改为对 dummy 哈希执行等价比对。
- **release 模式 CORS 通配**：原先仅 `slog.Warn` 不阻断启动。升级为 `config.validate()` 拒绝启动。
- **CORS 缓存污染**：回显具体 Origin 时未下发 `Vary: Origin`，前置代理的响应缓存会把为 A 域计算的 ACAO 命中给 B 域。

**部署与工具链**

- **`.env` 两套互斥变量名**：`JWT_SECRET`/`DB_PASSWORD`（生产脚本）与 `YF_AUTH_JWT_SECRET`/`YF_DATABASE_PASSWORD`
  （文档/Makefile）互斥 → **照文档配置仍启动失败**。统一为 `YF_` 前缀（`.env.example`/`start.sh`/`start.bat`/prod compose 同步）。
- **prod 日志格式**：生产 compose 未设 `YF_LOG_FORMAT`，走默认 `text`，而 Promtail/Loki 按 JSON 采集 → 日志聚合**静默降级**。已补。
- **`make db-migrate` 漏传口令**：从不设 `PGPASSWORD`，对有口令的库直接 `fe_sendauth` 失败。已补导出链。
- **CI lint 门槛失效**：`golangci-lint v1.62.2` 早于 Go 1.24，其内嵌 `go/types` 无法解析新标准库。升至 `v2.1.6`。
- **K8s 零 securityContext**：api 与 web 均补 `runAsNonRoot`/`readOnlyRootFilesystem`/`capabilities.drop`/`seccompProfile`；
  web 容器补 livenessProbe。

**测试**

- 新增回归测试 16 个用例 / 6 个文件：调拨负数、红冲净额归零、分次退药终态、自动拆零退货、药品停用、
  局部更新保角色、审核痕迹、预警处置、收货质检；效期 date-only、过期锁定、软删唯一性、JWT 吊销、
  空切片、并发 upsert；结算归集不串号、手工药品费进账、处方不重复计费、报表单一凭证、费用行回源；
  多副本单号唯一性（20 分配器 × 10 单号实测零重复）、格式列宽、号段不相交；
  `days=0` 绕过、超限量、最严归类、非法类型、必填项；限速边界、令牌吊销、`alg=none`、release CORS 门禁。
- **新增前后端字段契约守卫**（`pharmacy-web/src/types/contract.test.ts`，4 例）：
  扫描 `.vue` 中静态绑定的 `el-table-column prop="X"`，校验 X 存在于后端 struct 的 json tag 中。
  契约源取 `internal/` 而非 `docs/swagger.json`——多数 handler 走泛型 `Body` 返回，
  swag 不生成 definition，用 swagger 会产生 61 个误报使守卫失去意义。
- **修复测试基础设施缺陷**：`setupTestDB` 此前每用例各自 `OpenDB` 且从不关闭，用例数一多累计连接数超 `max_connections`，
  后续用例随机报 "too many clients already"，把基础设施问题伪装成业务失败。改为全测试共用单句柄。
- **修复集成测试会抹掉开发库数据**：`setupTestDB` 逐表 `TRUNCATE ... RESTART IDENTITY`，
  而库名硬编码为开发库 `yaofang`——跑一次集成测试即清空开发者本地数据。
  改为默认连独立库 `yaofang_test`（`YF_TEST_DB_NAME` 可覆盖），新增 `make test-db-init` 建库并迁移（幂等）。
- 迁移守护 `TestCleanupCoversAllTables` 由本轮自动抓出 `doc_segments` 漏登记并强制补齐。

**收尾修复（补契约测试时连带查出）**

- **效期/库存预警三列空白**：`StockAlertRepo.List` 是裸 `Find(model.StockAlert{})`，前端却渲染
  `drug_name` / `location_name` / `days_left`——与本轮修过的库存主列表同一类断裂，预警路径当时漏掉了。
  新增 `StockAlertRow` 行 DTO 并 JOIN drugs + inventory_locations，服务层签名同步。
  修复中踩到 SQL 陷阱并**实测确认**：PG 中 `date - date` 返回 `integer` 而非 `interval`，
  `EXTRACT(DAY FROM ...)` 会因缺少 `extract(unknown, integer)` 重载让整个接口运行期报错；已改用直接相减。
- **两个 CI 流水线从未触发**：`ci.yml` 与 `frontend-ci.yml` 的 `on.push.branches` 都只写 `[main]`，
  而本仓库主干是 `master` 且直接推 master → 迁移、lint、单测、集成、覆盖率、Swagger 一致性
  及全部前端门禁长期未执行，「CI 全绿」形同虚设。已改为 `[master, main]`。
- **`golangci-lint` 门禁配置与所锁版本不兼容**：`.golangci.yml` 是 v1 格式，CI 锁定 v2.1.6，
  v2 要求 `version: "2"`，直接报 `unsupported version of the configuration: ""` 并退出——
  **该门禁从未真正执行过**。已手写迁移为 v2 格式（`disable-all`→`default: none`、
  `linters-settings`→`linters.settings`、`ignore-words`→`ignore-rules`、
  `exclude-dirs`→`exclusions.paths`，并移除已并入 staticcheck 的 `gosimple`）。
- **lint 门槛实际变严并暴露 3 处 QF1003**：v1 的 `staticcheck` 只含 SA\* 检查，v2 合并了 QF\*，
  查出 3 处 if-else 链可改为 tagged switch（`clinical2_service.go` ×2、`prescription_service.go` ×1），
  已改，`golangci-lint run` 现为 **0 issues**。
- **`format:check` 报的 4 个文件里藏着编码缺陷**：`src/api/prescriptions.ts` 与 `src/api/purchase.ts`
  的中文注释是 **GBK 字节**混在 UTF-8 仓库中。`format:check` 把它们报成「格式不合规」，
  但那根本不是格式问题——执行 `npm run format` 会把非法字节替换成 U+FFFD，**不可逆销毁注释**。
  已按 GBK→UTF-8 转换并逐行修复二次编码，注释内容全部找回。
- **编码损坏已造成实际功能后果**：`prescriptions.ts` 第 10 行的损坏吃掉换行，
  `batch_group?: string` 被并进上一行注释。该字段是后端真实字段
  （`model.PrescriptionItem.BatchGroup`）且 `create.vue:298` 正在发送，却因不在接口中而
  完全脱离类型检查。已恢复为独立字段，注释按后端模型原文对齐。
- **无扩展名点文件缺 `eol=lf`**：`.gitattributes` 按扩展名声明了 `*.ts`/`*.vue` 等，
  但漏了 `.prettierrc`；在 `core.autocrlf=true` 的 Windows 检出下会被转成 CRLF，
  `format:check` 恒失败。已补 `.prettierrc text eol=lf` 与 `.eslintrc* text eol=lf`。
- **新增源码编码守卫**（`contract.test.ts` +2 例）：用 `TextDecoder(fatal)` 严格校验
  `internal/` 与 `pharmacy-web/src/` 下所有文本文件均为合法 UTF-8，并断言 `.gitattributes`
  声明了 `.prettierrc eol=lf`。反向验证：植入一个 GBK 探针文件后测试确实变红并指出文件名。
- 同步文档：docs/00 补 3 条坑位（测试库隔离、DATE 相减禁套 EXTRACT、主干分支名）、
  docs/06 新增 §8.1「前后端字段契约（强制）」、docs/10 补 §11.1 本地跑集成测试。

**推翻的静态审计结论（3 条，避免后人重复走弯路）**

- `InventorySummary` 的 `JOIN drugs` 冗余 → **不冗余**（CTE 取 `pack_size` 折算 LDU，外层取 `generic_name`）
- `supplier_repo.SetDefault` 的 `Transaction` 未绑定 ctx → **已绑定**（`WithContext` 会传递到 tx）
- `ExpiryAnalysis` 未按 `status=1` 过滤、`JOIN drugs` 未滤 `deleted_at` 属缺陷 → **属语义选择**
  （效期分析本就要包含已过期/已锁定批次以暴露风险；历史报表保留软删药名通常是期望行为）

**迁移 000037–000041**（共 41 版）：`qc_result` 去默认值 / 软删部分唯一索引 + 17 表 `deleted_at` 索引 /
`users.token_version` / `charge_records.visit_id` + `charge_items.source_record_id` / `doc_segments` 号段表。

**验证**：41 个迁移在干净 PG16 库顺序执行全通过；`gofmt`/`go build`/`go vet`（含 integration tag）全绿；
`golangci-lint v2.1.6` **0 issues**（配置迁到 v2 格式后才真正跑起来）；
单元测试 11 包全通过；集成测试（真实 PG16，独立测试库 `yaofang_test`）全通过；`docker build` 实测成功；
构建产物端到端（`/readyz`/登录/库存/报表）通过，令牌吊销实测「改密后旧 token 立即 401」；
覆盖率门槛全达标（domain 90.8 / middleware 66.7 / auth 85.7 / money 89.3 / config 39.6 /
service 44.7 / repository 31.3 / handler 12.3）；前端 `type-check`/`lint`/`format:check`/`test`
全绿（22 用例）；`swag` 后 `docs/` 无变化（符合预期）。
其中 Docker 构建、FEFO 口径、自动拆零退货、预警行 DTO（缺 JOIN 与套 EXTRACT 两种形态）、
契约守卫、编码守卫 六项做了**反向验证**（还原修复后测试确实失败并报出预期症状）。

## [v1.5.0] - 2026-09-28

> 软件著作权登记版本：累积第三/四/五轮审计修复与业务完善（S7 报表、CSV 导出、打印、站内通知、日志归档、迁移 33-36）。

### 新增（第五轮：业务与运维完善）

> 完整报告见 [docs/27-第五轮业务完善报告.md](docs/27-第五轮业务完善报告.md)。
> 医保真实结算保留 `port` 接口暂不实现；CI 流水线与压力测试留待另一台机器执行。

- **S7 合并结算报表（docs/20 收尾）**：`GET /reports/visit-volume`（按日挂号/结束/退号）、
  `/reports/revenue-breakdown`（已收费口径按费用项聚合，挂号/诊疗/药费占比）、
  `/reports/diagnosis-distribution`（Top20）；软删除表（visits/charges/medical_records）均带
  `deleted_at IS NULL`；前端报表页 3 新 Tab + ECharts。
- **报表导出 CSV**：`GET /reports/export?name=` 覆盖 9 报表，UTF-8 BOM（Excel 直接打开不乱码），
  金额分→元；前端每 Tab「导出CSV」按钮（blob 下载专用通道，不走信封拦截器）。
- **收货/发药库房可配置（000035）**：`default_receive_location`/`default_dispense_location`
  （种子 1/2）；`SettingService.LocationID` 缺失/非法/停用时回退编译默认值（发药/收货不中断），
  `Update` 强校验库房存在且启用；退药回补按发药记录溯源原库存行（库房变更后仍账实一致）。
- **操作日志归档（000035）**：`operation_logs_archive` 表 + `ArchiveBefore` 单语句 CTE 原子搬运
  （分批 5000/次）+ `RetentionDays`/`CleanupOldLogs` + 调度器每日 03:00
 （`scheduler.log_archive_cron`，保留期 `log_retention_days` 默认 180 天，0=不归档）。
- **站内通知（000036）**：`notifications` + `notification_reads`（已读按用户隔离，广播一人已读不影响他人）；
  `/notifications` 系列 4 端点（Authed 组）；调度器过期锁定 critical 广播 + 每日预警摘要 warning 广播（同日幂等）；
  前端顶栏铃铛（60s 轮询未读 + 下拉已读/全部已读）。
- **单据打印**：`pharmacy-web/src/utils/print.ts`（新窗口精简版式）+ 处方签/收费结算单/盘点单三处打印按钮。
- **前端版本对齐**：`pharmacy-web` 0.1.0 → 1.4.0（对齐后端发布基线）。
- **测试守护**：`testCleanupTables` 包变量化并补全至 40 表；`TestCleanupCoversAllTables`
  强制新增表登记（静态表附原因）；`report_export_test.go`（CSV/BOM/保留期解析）；
  `report_s7_sql_test.go`（归档 CTE 原子性断言；Raw+Scan 的 SELECT 不支持 DryRun，口径由集成环境覆盖）。

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
