// Package seq 生成业务单号：前缀 + 时间戳 + 号段内绝对序号，多副本部署下全局唯一。
//
// 背景：原实现为「前缀 + Unix 秒 + 进程内 3 位自增」，只保证**单进程内**唯一。
// 同一秒内若两个副本各自产生第 N 条单号，即产生完全相同的单号，撞 UNIQUE 约束；
// 而 PostgreSQL 下唯一冲突会把事务置入 aborted(25P02)，整笔业务（含库存扣减）回滚。
//
// 本实现改为「号段表」：doc_segments 按前缀维护一个单调递增的绝对序号。
// 每次用一条 INSERT ... ON CONFLICT DO UPDATE ... RETURNING 原子申请一段号
// （默认 256 个），该区间由本进程独占，因此不同副本拿到的区间必然不相交 → 全局唯一。
// DB 往返被摊薄到 1/blockSize。
//
// 不选其它方案的原因：
//
//	· nextval() 每单一次往返，比号段表贵 10~20 倍；
//	· Snowflake / 实例 ID 后缀依赖稳定实例身份，K8s 中 POD 名随重启变化，
//	  且跨节点时钟偏移会直接破坏单调性；
//	· UUID 破坏 VARCHAR(30) 列宽约束，且完全丧失「单号可按时间排序」这一药房现场刚需。
//
// 单号格式：<prefix><yyyyMMddHHmmss><8 位绝对序号>。
// 与旧格式（前缀 + 10 位秒 + 3 位序号 = 前缀后 13 位）长度不同，不会与存量单号冲突。
package seq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DefaultBlockSize 默认号段长度。取值权衡：
// 段越大 → DB 往返越少，但进程崩溃时该段作废、可用单号出现空洞。
// 药房单据允许编号不连续（不可重排），故偏向取大值以降低 DB 压力。
const DefaultBlockSize int64 = 256

// Allocator 号段分配器。Init 注入后为进程级单例使用。
type Allocator struct {
	db        *gorm.DB
	blockSize int64

	mu   sync.Mutex
	segs map[string]*seg
}

// seg 单个前缀的本地号段。[next, end) 为本进程独占区间。
type seg struct {
	next int64
	end  int64
}

var (
	defaultAlloc   *Allocator
	defaultAllocMu sync.RWMutex
)

// NewAllocator 构造独立分配器（blockSize <= 0 时用 DefaultBlockSize）。
// 正常服务路径使用包级单例（Init + Next）；本构造用于测试模拟多副本场景——
// 每个副本持有独立分配器，共享同一 doc_segments 表。
func NewAllocator(db *gorm.DB, blockSize int64) *Allocator {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}
	return &Allocator{db: db, blockSize: blockSize, segs: make(map[string]*seg)}
}

// Init 注入数据库连接并设置号段长度（blockSize <= 0 时用 DefaultBlockSize）。
// 必须在服务启动、数据库连接建立之后调用一次。
func Init(db *gorm.DB, blockSize int64) {
	defaultAllocMu.Lock()
	defaultAlloc = NewAllocator(db, blockSize)
	defaultAllocMu.Unlock()
}

func current() (*Allocator, error) {
	defaultAllocMu.RLock()
	a := defaultAlloc
	defaultAllocMu.RUnlock()
	if a == nil {
		return nil, fmt.Errorf("seq 未初始化：请在启动时调用 seq.Init(db, 0)")
	}
	return a, nil
}

// Next 生成形如 <prefix><yyyyMMddHHmmss><8 位绝对序号> 的单号。
//
// 返回错误仅在号段耗尽且重新申请失败时发生——此时宁可让调用方失败，
// 也绝不静默产生可能重复的单号。
func Next(ctx context.Context, prefix string) (string, error) {
	a, err := current()
	if err != nil {
		return "", err
	}
	return a.Next(ctx, prefix)
}

// Next 用指定分配器生成单号（便于测试注入）。
func (a *Allocator) Next(ctx context.Context, prefix string) (string, error) {
	a.mu.Lock()
	st, ok := a.segs[prefix]
	if !ok || st.next >= st.end {
		a.mu.Unlock()
		// 锁外申请：DB 往返不应阻塞其它前缀的号发放
		if err := a.allocBlock(ctx, prefix); err != nil {
			return "", err
		}
		a.mu.Lock()
		st = a.segs[prefix]
	}
	v := st.next
	st.next++
	a.mu.Unlock()
	return fmt.Sprintf("%s%s%08d", prefix, time.Now().Format("20060102150405"), v), nil
}

// allocBlock 原子申请一个号段。单条语句完成「不存在则插入 / 已存在则累加」，
// 并用 RETURNING 取回新的上界；PostgreSQL 下该语句是原子的，天然免并发冲突。
func (a *Allocator) allocBlock(ctx context.Context, prefix string) error {
	var next int64
	err := a.db.WithContext(ctx).Raw(`
		INSERT INTO doc_segments (prefix, next_val) VALUES (?, ?)
		ON CONFLICT (prefix) DO UPDATE
		   SET next_val = doc_segments.next_val + EXCLUDED.next_val,
		       updated_at = NOW()
		RETURNING next_val`, prefix, a.blockSize).Scan(&next).Error
	if err != nil {
		return fmt.Errorf("申请单号段失败(prefix=%s): %w", prefix, err)
	}
	a.mu.Lock()
	a.segs[prefix] = &seg{next: next - a.blockSize, end: next}
	a.mu.Unlock()
	return nil
}
