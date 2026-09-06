# ── 药房管理系统 前端镜像（Vite 构建 → Nginx 托管 SPA + /api 反代）──
# 用法：
#   docker build -t yaofang-web pharmacy-web
# 运行时通过环境变量 API_UPSTREAM 指定后端地址（compose 内按服务名注入，如 api:8080 / backend:8080）。
# ────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

# 国内网络加速（海外环境可删除）
RUN npm config set registry https://registry.npmmirror.com

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# 生产 API 基地址：默认同源 /api/v1（Nginx 反代 /api → 后端）
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

# ── 运行时：Nginx ──
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
# 模板中 ${API_UPSTREAM} 由官方镜像入口 envsubst 在容器启动时替换
COPY nginx.conf /etc/nginx/templates/default.conf.template

EXPOSE 80
