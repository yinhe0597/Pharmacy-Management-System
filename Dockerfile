# ── 药房管理系统 后端镜像（多阶段构建）──
# 用法：
#   docker build -t yaofang-api .
#   docker build --build-arg VERSION=v1.4.0 --build-arg COMMIT=$(git rev-parse --short HEAD) -t yaofang-api .
# 运行时全部配置走环境变量（YF_ 前缀），镜像内不含任何配置文件/密钥。
# ────────────────────────────────────────
FROM golang:1.26.5-alpine AS builder

WORKDIR /src

# 国内网络加速（海外环境可删除）
ENV GOPROXY=https://goproxy.cn,direct

# 依赖缓存层：go.mod/go.sum 变更才失效
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDTIME=unknown

# CGO_ENABLED=0 静态编译，-trimpath 去除构建路径，-ldflags 注入版本信息
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w \
      -X yaofang/internal/version.Version=${VERSION} \
      -X yaofang/internal/version.Commit=${COMMIT} \
      -X yaofang/internal/version.BuildTime=${BUILDTIME}" \
    -o /out/yaofang ./cmd/server

# ── 运行时：Alpine 精简镜像（非 root 运行；时区已内嵌 time/tzdata，无需额外 tzdata 包）──
FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -S yaofang && adduser -S -G yaofang yaofang

COPY --from=builder /out/yaofang /usr/local/bin/yaofang

USER yaofang
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/yaofang"]
