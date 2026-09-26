#!/usr/bin/env bash
# 覆盖率门槛检查（docs/07 §7）：
#   1) 单元测试口径：domain 四包 ≥85%、middleware ≥30%、pkg/* ≥60%（无需数据库）
#   2) 集成测试口径（需 PostgreSQL）：service ≥35%、repository ≥20%、handler ≥10%
#      —— 集成用例位于 service/handler 包（build tag integration），以 -coverpkg 统计被测包真实覆盖。
#
# 用法：
#   scripts/check_coverage.sh                # 两段都跑（需 make db-up 起库）
#   YF_SKIP_INTEGRATION=1 scripts/check_coverage.sh   # 只跑单元口径（CI 之外的快速检查）
#
# 门槛值取自当前实测水位并留有余量（改动后若覆盖率上升，可上调门槛）。
set -uo pipefail

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail=0

# pkg_cov <profile> <package-prefix>：按语句数加权计算该包覆盖率（百分比，一位小数）
pkg_cov() {
  awk -v pkg="$2" '
    NR > 1 && index($1, pkg) == 1 {
      total += $2
      if ($3 + 0 > 0) covered += $2
    }
    END {
      if (total == 0) { print "0.0" } else { printf "%.1f", covered * 100 / total }
    }
  ' "$1"
}

# gate <名称> <实测值> <门槛>
gate() {
  local name="$1" got="$2" min="$3"
  if awk -v g="$got" -v m="$min" 'BEGIN { exit !(g + 0 >= m + 0) }'; then
    printf '  ✓ %-42s %6s%%  (门槛 ≥%s%%)\n' "$name" "$got" "$min"
  else
    printf '  ✗ %-42s %6s%%  (门槛 ≥%s%%)\n' "$name" "$got" "$min"
    fail=1
  fi
}

echo "== 单元测试口径 =="
go test -count=1 -covermode=set -coverprofile="$TMP/unit.out" ./internal/... >/dev/null 2>&1 || {
  echo "单元测试执行失败，请先运行 go test ./internal/..." >&2
  exit 1
}
# domain 四包合并口径（与历史门槛一致）
: > "$TMP/domain.out"
printf 'mode: set\n' > "$TMP/domain.out"
grep -v '^mode:' "$TMP/unit.out" | grep -E 'internal/domain/(enum|prescription|rule|interaction)/' >> "$TMP/domain.out"
gate "domain（enum/prescription/rule/interaction）" "$(pkg_cov "$TMP/domain.out" "yaofang/internal/domain")" 85
gate "internal/middleware" "$(pkg_cov "$TMP/unit.out" "yaofang/internal/middleware")" 30
gate "internal/pkg/auth" "$(pkg_cov "$TMP/unit.out" "yaofang/internal/pkg/auth")" 60
gate "internal/pkg/money" "$(pkg_cov "$TMP/unit.out" "yaofang/internal/pkg/money")" 60
gate "internal/config" "$(pkg_cov "$TMP/unit.out" "yaofang/internal/config")" 30

if [ "${YF_SKIP_INTEGRATION:-0}" = "1" ]; then
  echo "（已跳过集成口径：YF_SKIP_INTEGRATION=1）"
else
  echo "== 集成测试口径（真实 PostgreSQL）=="
  # 注意：-coverpkg 需精确限定到目标包——若写成 ./internal/...（包含测试包自身），
  # Go 的覆盖统计会因同一包被「被测包 + coverpkg」双重插桩而使语句数翻倍、百分比失真。
  if ! go test -tags=integration -count=1 -covermode=set \
      -coverpkg=./internal/service/...,./internal/repository/... \
      -coverprofile="$TMP/integ_svc.out" ./internal/service/ >"$TMP/integ_svc.log" 2>&1; then
    echo "集成测试执行失败（需 PostgreSQL：make db-up）：" >&2
    tail -20 "$TMP/integ_svc.log" >&2
    exit 1
  fi
  gate "internal/service" "$(pkg_cov "$TMP/integ_svc.out" "yaofang/internal/service/")" 35
  gate "internal/repository" "$(pkg_cov "$TMP/integ_svc.out" "yaofang/internal/repository/")" 20

  if ! go test -tags=integration -count=1 -covermode=set \
      -coverpkg=./internal/handler/ \
      -coverprofile="$TMP/integ_handler.out" ./internal/handler/ >"$TMP/integ_handler.log" 2>&1; then
    echo "HTTP 层集成测试执行失败（需 PostgreSQL）：" >&2
    tail -20 "$TMP/integ_handler.log" >&2
    exit 1
  fi
  gate "internal/handler（HTTP 层）" "$(pkg_cov "$TMP/integ_handler.out" "yaofang/internal/handler/")" 10
fi

echo
if [ "$fail" -eq 0 ]; then
  echo "覆盖率门槛全部通过 ✅"
else
  echo "覆盖率门槛未通过 ❌" >&2
  exit 1
fi
