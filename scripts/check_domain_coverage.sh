#!/usr/bin/env bash
# domain 包覆盖率门槛（docs/07 §7：domain ≥ 85%）
# 用法: scripts/check_domain_coverage.sh
set -euo pipefail

THRESHOLD=85
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "mode: set" > "$TMP/merged.out"
for pkg in ./internal/domain/enum ./internal/domain/prescription ./internal/domain/rule ./internal/domain/interaction; do
  go test -coverprofile="$TMP/pkg.out" "$pkg" >/dev/null
  grep -v '^mode:' "$TMP/pkg.out" >> "$TMP/merged.out"
done

total=$(go tool cover -func="$TMP/merged.out" | tail -1 | awk '{print $NF}' | tr -d '%')
echo "domain 覆盖率: ${total}%"
awk -v t="$total" -v th="$THRESHOLD" 'BEGIN { if (t < th) { print "domain 覆盖率不足 " th "%"; exit 1 } else { print "覆盖率门槛通过" } }'
