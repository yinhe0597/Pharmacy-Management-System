// Package seq 生成业务单号：前缀 + 时间戳 + 进程内自增序号，保证单实例内唯一。
package seq

import (
	"fmt"
	"sync"
	"time"
)

var (
	mu            sync.Mutex
	lastTimestamp int64
	counter       int
)

// Next 生成形如 <prefix><yyyymmddHHMMSS><3位序号> 的单号。
func Next(prefix string) string {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().Unix()
	if now == lastTimestamp {
		counter++
	} else {
		lastTimestamp = now
		counter = 0
	}
	return fmt.Sprintf("%s%d%03d", prefix, now, counter)
}
