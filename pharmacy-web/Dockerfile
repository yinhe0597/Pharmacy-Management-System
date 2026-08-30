# 药房管理系统前端镜像：构建静态文件 + Nginx 托管并反代后端 API
FROM node:20-alpine AS build
WORKDIR /web
# 国内网络加速（海外环境可删除）
RUN npm config set registry https://registry.npmmirror.com
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /web/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
