# 药房管理系统后端镜像（多阶段构建）
# 构建：docker build -t yaofang-backend .
FROM golang:1.26-alpine AS build
WORKDIR /src
# 国内网络加速（海外环境可删除）
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/yaofang ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /out/yaofang ./yaofang
EXPOSE 8080
# 配置全部通过 YF_ 前缀环境变量注入（如 YF_DATABASE_HOST / YF_AUTH_JWT_SECRET）
ENTRYPOINT ["/app/yaofang"]
