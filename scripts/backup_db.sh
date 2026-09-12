#!/usr/bin/env bash
# 药房管理系统 · PostgreSQL 备份脚本（pg_dump 自定义格式 + 保留策略）
#
# 用法：
#   ./scripts/backup_db.sh                          # 使用默认连接（localhost:5432/yaofang）
#   BACKUP_DIR=/backup RETENTION_DAYS=30 ./scripts/backup_db.sh
#   生产一键部署（容器内 db 服务）自动识别：找不到 psql 时回退 docker compose exec
#
# 定时任务（crontab -e，每日 02:30，日志见 /var/log/yaofang-backup.log）：
#   30 2 * * * cd /opt/yaofang && ./scripts/backup_db.sh >> /var/log/yaofang-backup.log 2>&1
#
# 恢复：
#   pg_restore -h <host> -U yaofang -d yaofang --clean --if-exists /backup/yaofang_YYYYmmdd_HHMMSS.dump
set -euo pipefail
cd "$(dirname "$0")/.."

# 自动读取部署目录下的 .env（一键部署由 start.sh/start.bat 生成，含 YF_DATABASE_PASSWORD），
# 使 `./scripts/backup_db.sh` 在无需手工导出环境变量的情况下即可工作。
if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

DB_HOST="${YF_DB_HOST:-${YF_DATABASE_HOST:-localhost}}"
DB_PORT="${YF_DB_PORT:-${YF_DATABASE_PORT:-5432}}"
DB_USER="${YF_DB_USER:-${YF_DATABASE_USER:-yaofang}}"
DB_NAME="${YF_DB_NAME:-${YF_DATABASE_NAME:-yaofang}}"
DB_PASSWORD="${YF_DB_PASSWORD:-${YF_DATABASE_PASSWORD:-${PGPASSWORD:-}}}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"

mkdir -p "$BACKUP_DIR"
umask 077
STAMP="$(date +%Y%m%d_%H%M%S)"
OUT="$BACKUP_DIR/yaofang_${STAMP}.dump"

backup_ok=0

# 路径一：本机 pg_dump（-w 禁止交互式口令提示：口令缺失/库不可达时立即失败并回退，
# 避免脚本在无人值守的 crontab 中永久挂起）
if command -v pg_dump >/dev/null 2>&1; then
  echo "[backup] 尝试 pg_dump → $OUT（$DB_USER@$DB_HOST:$DB_PORT/$DB_NAME）"
  if PGPASSWORD="$DB_PASSWORD" pg_dump -w \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -Fc --no-owner -f "$OUT"; then
    backup_ok=1
  else
    echo "[backup] pg_dump 失败（口令缺失/库不可达），尝试回退容器内备份..."
    rm -f "$OUT"
  fi
fi

# 路径二：容器内 pg_dump（一键部署形态：db 服务在 compose 中运行，容器内为本地信任认证）
if [ "$backup_ok" -eq 0 ] && command -v docker >/dev/null 2>&1; then
  db_cid="$(docker compose -f "$COMPOSE_FILE" ps -q db 2>/dev/null || true)"
  if [ -n "$db_cid" ]; then
    echo "[backup] docker compose exec db pg_dump → $OUT"
    if docker compose -f "$COMPOSE_FILE" exec -T db \
      pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc --no-owner > "$OUT"; then
      backup_ok=1
    else
      rm -f "$OUT"
    fi
  fi
fi

if [ "$backup_ok" -eq 0 ]; then
  echo "[backup] 错误：未能完成备份。" >&2
  echo "[backup] 请安装 postgresql-client 并提供口令（YF_DATABASE_PASSWORD），" >&2
  echo "[backup] 或在部署目录内确保 $COMPOSE_FILE 的 db 服务正在运行。" >&2
  exit 1
fi

if [ ! -s "$OUT" ]; then
  echo "[backup] 错误：备份文件为空，已删除：$OUT" >&2
  rm -f "$OUT"
  exit 1
fi

echo "[backup] 完成：$OUT（$(du -h "$OUT" | cut -f1)）"
echo "[backup] 清理 ${RETENTION_DAYS} 天前的旧备份..."
find "$BACKUP_DIR" -name 'yaofang_*.dump' -type f -mtime "+${RETENTION_DAYS}" -print -delete
echo "[backup] 全部完成。"
