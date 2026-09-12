#!/bin/sh
# 数据库迁移执行器：带版本表（schema_migrations）的**增量**迁移，可安全重复执行。
#
# 背景：原实现按文件名顺序重放全部 migrations/*.up.sql，只在「全新数据库」上成立；
# 对已初始化的库再次执行会因 CREATE TABLE 已存在而失败（生产二次部署/升级会因此阻断）。
#
# 用法（连接参数经环境变量注入，与 postgres 官方镜像一致）：
#   PGHOST=db PGUSER=yaofang PGPASSWORD=*** PGDATABASE=yaofang sh scripts/migrate.sh
#   MIGRATIONS_DIR=/migrations        # 迁移目录（默认 /migrations）
#   YF_MIGRATE_BASELINE_TO=000032     # 仅对「历史库」一次性使用：把 ≤ 该版本的迁移标记为已应用
#
# 行为：
#   1) 确保 schema_migrations 版本表存在；
#   2) 全新库（无 users 表）→ 按序执行全部未应用迁移；
#   3) 已跟踪的库 → 仅执行未应用的迁移（重复运行输出 skip，幂等）；
#   4) 历史库（有 users 表但无版本记录）→ 拒绝自动猜测，要求显式指定
#      YF_MIGRATE_BASELINE_TO（避免「已应用的迁移被重放」或「未应用的迁移被跳过」）。
set -eu

PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-yaofang}"
PGDATABASE="${PGDATABASE:-yaofang}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-/migrations}"
BASELINE_TO="${YF_MIGRATE_BASELINE_TO:-}"

if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "错误：迁移目录不存在: $MIGRATIONS_DIR" >&2
  exit 1
fi

echo "==> 迁移开始 (host=${PGHOST}, user=${PGUSER}, db=${PGDATABASE}, dir=${MIGRATIONS_DIR})"

psql -v ON_ERROR_STOP=1 -q -c "CREATE TABLE IF NOT EXISTS schema_migrations (
    version    VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)"

applied_count=$(psql -v ON_ERROR_STOP=1 -tAc "SELECT count(*) FROM schema_migrations")
users_exists=$(psql -v ON_ERROR_STOP=1 -tAc "SELECT to_regclass('public.users') IS NOT NULL")

# 历史库引导：库已被旧版（全量重放）方式初始化，但没有版本记录
if [ "$applied_count" = "0" ] && [ "$users_exists" = "t" ]; then
  if [ -z "$BASELINE_TO" ]; then
    echo "错误：检测到已初始化的历史库（存在 users 表）但没有迁移版本记录。" >&2
    echo "      请一次性指定基线版本后重试，例如（当前已部署到 000032 时）：" >&2
    echo "        YF_MIGRATE_BASELINE_TO=000032 sh scripts/migrate.sh" >&2
    echo "      该参数会把 ≤ 基线版本的迁移标记为已应用，随后仅执行更新的迁移。" >&2
    exit 1
  fi
  echo "==> 历史库引导：将 ≤ ${BASELINE_TO} 的迁移标记为已应用（不执行 SQL）"
  for f in $(ls "$MIGRATIONS_DIR"/*.up.sql | sort); do
    name=$(basename "$f")
    ver=$(echo "$name" | cut -d_ -f1)
    if [ "$ver" \> "$BASELINE_TO" ]; then
      continue
    fi
    psql -v ON_ERROR_STOP=1 -q -c \
      "INSERT INTO schema_migrations (version) VALUES ('$name') ON CONFLICT DO NOTHING"
    echo "    baseline $name"
  done
fi

pending=0
for f in $(ls "$MIGRATIONS_DIR"/*.up.sql | sort); do
  name=$(basename "$f")
  if [ "$(psql -v ON_ERROR_STOP=1 -tAc "SELECT 1 FROM schema_migrations WHERE version = '$name'")" = "1" ]; then
    echo "== skip $name"
    continue
  fi
  echo "== apply $name"
  # 单文件单事务：失败即整体回滚，不留下半应用状态
  psql -v ON_ERROR_STOP=1 -q --single-transaction -f "$f"
  psql -v ON_ERROR_STOP=1 -q -c "INSERT INTO schema_migrations (version) VALUES ('$name')"
  pending=$((pending + 1))
done

echo "==> 迁移完成（本次应用 $pending 个；版本表记录 $(psql -v ON_ERROR_STOP=1 -tAc 'SELECT count(*) FROM schema_migrations') 个）"
