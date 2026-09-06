#!/bin/sh
# 数据库迁移入口：按文件名顺序执行全部 migrations/*.up.sql。
# 连接参数经环境变量注入：PGHOST / PGUSER / PGPASSWORD / PGDATABASE。
set -e

echo "==> 开始执行迁移 (host=${PGHOST:-localhost}, db=${PGDATABASE:-yaofang})"
for f in /migrations/*.up.sql; do
  echo "== $f"
  psql -v ON_ERROR_STOP=1 -f "$f"
done
echo "==> 迁移全部完成"
