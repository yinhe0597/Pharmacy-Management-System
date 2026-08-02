# 更新日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 与 [Keep a Changelog](https://keepachangelog.com/zh-CN/)。

## [v1.3.0] - 2026-08-02

### 基础参考数据（种子）

- **ICD-10 疾病诊断编码**（迁移 `000015`）：`diagnosis_codes` 表，1,586 条诊断编码，含 22 个一级章节 + 220 个二级分类的层级结构。数据来源 [ICD-10-CN](https://github.com/chaseliu/ICD-10-CN)。用于诊断名称模糊搜索/自动补全。
- **国家集采药品目录**（迁移 `000016`）：`vbp_drug_catalog` 表，392 个品种（去重），覆盖第 1-10 批国家药品集中带量采购全部批次（2018-2025）。含药品通用名、剂型、剂型分类、集采批次。用于药品名称模糊搜索/自动补全，后续入库时再精确填列规格/厂家/价格。

### 新增表

- `diagnosis_codes` — ICD-10 诊断编码查找表（code, disease_name, chapter_code/name, category_code/name, py_code）
- `vbp_drug_catalog` — 国家集采药品参考目录（generic_name, dosage_form, dosage_category, vbp_batch, py_code）

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

### 数据库迁移（6 个版本）

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
