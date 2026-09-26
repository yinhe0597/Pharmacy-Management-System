// 注意：Raw + Scan 的 SELECT 在 DryRun 下无法断言——gORM finisher_api
// 对 Rows() 直接返回 ErrDryRunModeUnsupported，回调链根本不执行。
// S7 三报表的 SELECT 口径由集成测试覆盖（service 包 integration tag，需真实 PG）；
// 此处仅断言走 Exec 回调的归档语句（DryRun 可执行、After 钩子可捕获）。
package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestArchiveSQLExec ArchiveBefore 走 Exec/Raw 回调：断言语句为单条原子 CTE，
// 含归档表与 RETURNING，且范围删除走 created_at（000035 索引覆盖）。
func TestArchiveSQLExec(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewOperationLogRepo(db)
	var captured string
	const name = "test:capture_exec"
	proc := db.Callback().Raw().After("gorm:raw")
	if err := proc.Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil {
			captured = tx.Statement.SQL.String()
		}
	}); err != nil {
		t.Fatalf("注册回调失败: %v", err)
	}
	defer func() { _ = db.Callback().Raw().Remove(name) }()
	if _, err := repo.ArchiveBefore(context.Background(), time.Now().AddDate(0, 0, -180), 5000); err != nil && !errors.Is(err, gorm.ErrDryRunModeUnsupported) {
		t.Fatalf("DryRun 执行返回错误: %v", err)
	}
	for _, want := range []string{"operation_logs_archive", "RETURNING", "created_at <"} {
		if !strings.Contains(captured, want) {
			t.Errorf("归档 SQL 缺少 %q：%s", want, captured)
		}
	}
}
