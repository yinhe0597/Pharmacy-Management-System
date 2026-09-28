//go:build integration

// 单号号段分配器（000041）回归测试：多副本/多分配器下全局唯一。
package service_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"yaofang/internal/pkg/seq"
)

// TestSeqUniqueAcrossConcurrentAllocators 多个「副本」各自持有独立分配器时，
// 产生的单号必须全局唯一。
//
// 回归：原实现为「前缀+Unix秒+进程内3位自增」，只在单进程内唯一；
// 同一秒内两个副本各自产生第 N 条即得到完全相同的单号，撞 UNIQUE 约束，
// 而 PostgreSQL 下唯一冲突会把事务置入 aborted(25P02)，整笔业务（含库存扣减）回滚。
func TestSeqUniqueAcrossConcurrentAllocators(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// 20 个「副本」，每个独立分配器（独立号段缓存，共享 doc_segments 表）
	// 号段取 32：每个副本只触发 1 次 DB 申请，20 路并发不会压垮测试库的连接数
	// （用 1~8 的小号段会在高并发下打爆 max_connections，那是测试自身的问题而非产品缺陷）。
	const replicas = 20
	const perReplica = 10
	allocs := make([]*seq.Allocator, replicas)
	for i := range allocs {
		allocs[i] = seq.NewAllocator(db, 32)
	}

	var mu sync.Mutex
	seen := make(map[string]string, replicas*perReplica)
	dups := make([]string, 0)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for r := 0; r < replicas; r++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			for i := 0; i < perReplica; i++ {
				no, err := allocs[idx].Next(ctx, "RX")
				if err != nil {
					t.Errorf("生成单号失败: %v", err)
					return
				}
				mu.Lock()
				if prev, ok := seen[no]; ok {
					dups = append(dups, fmt.Sprintf("%s 同时被 %s 与 副本%d 产生", no, prev, idx))
				} else {
					seen[no] = fmt.Sprintf("副本%d", idx)
				}
				mu.Unlock()
			}
		}(r)
	}
	close(start)
	wg.Wait()

	if len(dups) > 0 {
		t.Fatalf("产生 %d 个重复单号，例：%s", len(dups), dups[0])
	}
	if len(seen) != replicas*perReplica {
		t.Fatalf("期望 %d 个唯一单号，实际 %d", replicas*perReplica, len(seen))
	}
}

// TestSeqFormatAndColumnWidth 单号格式与列宽约束：
// 格式 <prefix><yyyyMMddHHmmss><8 位序号>，总长须 ≤ 30（所有 *_no 列为 VARCHAR(30)）。
func TestSeqFormatAndColumnWidth(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	a := seq.NewAllocator(db, 16)

	// 覆盖全部在用前缀（含 3 字符前缀）
	for _, prefix := range []string{"V", "C", "RX", "PO", "RCV", "REQ", "STK", "SPL", "RSV", "ITN"} {
		no, err := a.Next(ctx, prefix)
		if err != nil {
			t.Fatalf("prefix=%s 生成单号失败: %v", prefix, err)
		}
		if !strings.HasPrefix(no, prefix) {
			t.Fatalf("单号 %q 应以 %q 开头", no, prefix)
		}
		if len(no) > 30 {
			t.Fatalf("单号 %q 长度 %d 超过列宽 VARCHAR(30)", no, len(no))
		}
		// prefix + 14 位时间戳 + 8 位序号
		wantLen := len(prefix) + 22
		if len(no) != wantLen {
			t.Fatalf("prefix=%s 单号 %q 长度 %d，期望 %d", prefix, no, len(no), wantLen)
		}
		// 8 位序号部分必须为数字
		tail := no[len(prefix)+14:]
		for _, c := range tail {
			if c < '0' || c > '9' {
				t.Fatalf("单号 %q 序号部分含非数字字符 %q", no, c)
			}
		}
	}
}

// TestSeqAllocatesDisjointBlocks 不同分配器拿到不相交的号段：
// 这是跨副本唯一的根本保证（即使同一秒内产生大量单号也不会重复）。
func TestSeqAllocatesDisjointBlocks(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	const block = 4

	a1 := seq.NewAllocator(db, block)
	a2 := seq.NewAllocator(db, block)

	// a1 先取，走满整段
	var a1nos []string
	for i := 0; i < block; i++ {
		no, err := a1.Next(ctx, "ITN")
		if err != nil {
			t.Fatalf("a1 生成失败: %v", err)
		}
		a1nos = append(a1nos, no)
	}
	// a2 再取，应拿到 a1 之后的号段
	var a2nos []string
	for i := 0; i < block; i++ {
		no, err := a2.Next(ctx, "ITN")
		if err != nil {
			t.Fatalf("a2 生成失败: %v", err)
		}
		a2nos = append(a2nos, no)
	}

	seqOf := func(no string) int64 {
		var v int64
		_, _ = fmt.Sscanf(no[len("ITN")+14:], "%d", &v)
		return v
	}
	maxA1 := seqOf(a1nos[len(a1nos)-1])
	minA2 := seqOf(a2nos[0])
	if minA2 <= maxA1 {
		t.Fatalf("a2 首个序号(%d) 应大于 a1 末个序号(%d)：号段必须不相交", minA2, maxA1)
	}
}
