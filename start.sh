#!/usr/bin/env bash
# 药房管理系统 · Linux 一键部署
set -e
cd "$(dirname "$0")"

if [ ! -f .env ]; then
  echo "[1/3] 首次运行：生成 .env（随机 JWT 密钥）..."
  JWT_SECRET=$(head -c 48 /dev/urandom | tr -dc 'A-Za-z0-9' | head -c 48)
  cat > .env <<EOF
JWT_SECRET=${JWT_SECRET}
DB_PASSWORD=yaofang123
HTTP_PORT=80
EOF
else
  echo "[1/3] .env 已存在，跳过生成"
fi

echo "[2/3] 构建并启动容器..."
docker compose -f docker-compose.prod.yml up -d --build

echo "[3/3] 等待服务就绪..."
for i in $(seq 1 30); do
  if curl -sf "http://localhost:${HTTP_PORT:-80}/healthz" >/dev/null 2>&1; then
    echo
    echo "============================================"
    echo " 部署完成！"
    echo " 本机访问:  http://localhost:${HTTP_PORT:-80}"
    echo " 内网访问:  http://<本机IP>:${HTTP_PORT:-80}"
    echo " 默认账号:  admin / admin123 （请立即改密码！）"
    echo "============================================"
    exit 0
  fi
  sleep 3
done
echo "服务仍在启动中，请稍后访问 http://localhost:${HTTP_PORT:-80}"
