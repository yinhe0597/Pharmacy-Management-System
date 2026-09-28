#!/usr/bin/env bash
# 药房管理系统 · Linux 一键部署
# 首次运行自动生成 .env：CSPRNG 随机 JWT 密钥（≥32 字符，后端启动校验要求）+ 随机数据库口令。
set -euo pipefail
cd "$(dirname "$0")"

# gen_secret <字节数>：生成密码学安全随机串。
# 注意：不可用 `head -c N /dev/urandom | tr -dc 'A-Za-z0-9'`——过滤后长度会缩水到约 N/4，
# 达不到后端 config.validate() 的 ≥32 字符要求，会导致后端容器启动即退出（crash-loop）。
gen_secret() {
  local bytes="${1:-48}"
  if command -v openssl >/dev/null 2>&1; then
    # tr 同时剥离 \r：Git Bash/Cygwin 下的 openssl 输出 CRLF，尾部 CR 会污染密钥
    openssl rand -base64 "$bytes" | tr -d '\r\n'
  else
    # base64 输出恒为可打印 ASCII：48 字节 → 约 64 字符
    head -c "$bytes" /dev/urandom | base64 | tr -d '\r\n'
  fi
}

# gen_hex <字节数>：生成十六进制随机串（数据库口令用：无特殊字符，避免 DSN/Compose 转义问题）。
gen_hex() {
  local bytes="${1:-24}"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$bytes" | tr -d '\r\n'
  else
    head -c "$bytes" /dev/urandom | od -An -tx1 | tr -d ' \r\n'
  fi
}

if [ ! -f .env ]; then
  echo "[1/4] 首次运行：生成 .env（随机 JWT 密钥 + 随机数据库口令）..."
  JWT_SECRET="$(gen_secret 48)"
  DB_PASSWORD="$(gen_hex 24)"
  if [ "${#JWT_SECRET}" -lt 32 ]; then
    echo "错误：JWT 密钥生成失败（长度 ${#JWT_SECRET} < 32），请检查 openssl 或 /dev/urandom 可用性。" >&2
    exit 1
  fi
  umask 077
  # 键名与 .env.example / Makefile env-init 统一为 YF_ 前缀：
  # 历史上的 JWT_SECRET/DB_PASSWORD 与 YF_* 两套命名互斥，照文档配置也会启动失败。
  cat > .env <<EOF
# 由 start.sh 于 $(date '+%Y-%m-%dT%H:%M:%S%z') 自动生成
# 含密钥与口令，请勿提交版本库、勿外传；重新生成会使既有登录 token 失效。
YF_AUTH_JWT_SECRET='${JWT_SECRET}'
YF_DATABASE_PASSWORD='${DB_PASSWORD}'
HTTP_PORT=80
EOF
  echo "      已生成 .env（权限 600）：数据库口令为随机值，可在 .env 中查看。"
else
  echo "[1/4] .env 已存在，跳过生成"
fi

# 读取 .env 以获取 HTTP_PORT（供就绪探测使用）
set -a
# shellcheck disable=SC1091
. ./.env
set +a

echo "[2/4] 校验部署配置..."
if [ -z "${YF_AUTH_JWT_SECRET:-}" ] || [ "${#YF_AUTH_JWT_SECRET}" -lt 32 ]; then
  echo "错误：.env 中 YF_AUTH_JWT_SECRET 缺失或不足 32 字符。请删除 .env 重新运行本脚本，或手动设置强随机密钥（openssl rand -base64 48）。" >&2
  exit 1
fi
if [ -z "${YF_DATABASE_PASSWORD:-}" ]; then
  echo "错误：.env 中 YF_DATABASE_PASSWORD 缺失。请删除 .env 重新运行本脚本。" >&2
  exit 1
fi

echo "[3/4] 构建并启动容器..."
docker compose -f docker-compose.prod.yml up -d --build

echo "[4/4] 等待服务就绪..."
for _ in $(seq 1 30); do
  if curl -sf "http://localhost:${HTTP_PORT:-80}/healthz" >/dev/null 2>&1; then
    echo
    echo "============================================"
    echo " 部署完成！"
    echo " 本机访问:  http://localhost:${HTTP_PORT:-80}"
    echo " 内网访问:  http://<本机IP>:${HTTP_PORT:-80}"
    echo " 默认账号:  admin / admin123（种子账号，请立即修改密码）"
    echo " 数据库口令: 随机生成，见 .env（YF_DATABASE_PASSWORD）"
    echo " ⚠ release 模式下若仍使用默认口令 admin123，服务将拒绝启动。"
    echo "============================================"
    exit 0
  fi
  sleep 3
done
echo "服务仍在启动中，请稍后访问 http://localhost:${HTTP_PORT:-80}"
echo "如持续失败请查看日志：docker compose -f docker-compose.prod.yml logs -f backend"
