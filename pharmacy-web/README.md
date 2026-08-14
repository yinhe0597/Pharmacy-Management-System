# 💊 药房管理系统 · 前端（pharmacy-web）

药房进销存与处方调配后端配套的前端管理台。技术栈：**Vite + Vue 3 + TypeScript + Element Plus + Pinia + Vue Router + Axios**。

> 设计与进度规划见 [../docs/17-前端开发指南与进度规划.md](../docs/17-前端开发指南与进度规划.md)，
> 对接约定见 [../docs/16-前端开发就绪评估与对接指南.md](../docs/16-前端开发就绪评估与对接指南.md)。

## 快速开始

```bash
# 1. 启动后端（另开终端，在项目根目录）
make db-up          # Docker 起库 + 迁移
go run ./cmd/server # http://localhost:8080

# 2. 前端
cd pharmacy-web
npm install
npm run dev         # http://localhost:5173，/api 已代理到 :8080
```

默认账号（密码均 `admin123`）：`admin` / `doctor` / `clinic_nurse` / `pharmacy_nurse` / `pharmacy_chief` / `pharmacist` / `buyer` / `finance`。

## 目录结构

```
src/
├── api/            # 按后端模块封装的接口层（http.ts 统一信封/错误/401）
├── types/          # 契约类型（api.ts 信封/分页；business.ts 角色/权限/状态枚举）
├── stores/         # Pinia（user：token/角色/权限）
├── router/         # 路由 + 守卫（登录/角色权限）；menu.ts 角色菜单
├── directives/     # v-permission 按钮级权限
├── layouts/        # MainLayout（侧边菜单/顶栏）
├── views/          # 页面（login/dashboard + 各域占位）
└── utils/          # auth(令牌)/money(分)/format
```

## 脚本

| 命令 | 说明 |
|------|------|
| `npm run dev` | 开发启动 |
| `npm run type-check` | TS 类型检查 |
| `npm run build` | 类型检查 + 生产构建（产物 `dist/`） |
| `npm run preview` | 预览构建产物 |

## 约定

- 金额一律「分」，仅展示层用 `utils/money.ts` 格式化；冲正/红冲为负金额。
- 权限：路由 `meta.permission` + 菜单 `menu.ts` + 按钮 `v-permission`，与后端角色矩阵一致。
- 接口信封 `{code,message,data}`，`code!==0` 由拦截器统一提示。

## 进度

- ✅ P0 工程骨架 + 基础设施 + 登录/布局/路由守卫
- ⏳ P1-P6 各域页面（按 docs/17 计划推进）
