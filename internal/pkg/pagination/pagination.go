// Package pagination 提供统一分页参数与结果结构。
package pagination

// Query 分页查询参数。page 从 1 起，page_size 默认 20，最大 200。
type Query struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// Result 分页响应结构。
type Result[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Normalize 规整分页参数到合法范围。
func (q *Query) Normalize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
}

// Offset 返回 SQL OFFSET。
func (q *Query) Offset() int { return (q.Page - 1) * q.PageSize }

// Limit 返回 SQL LIMIT。
func (q *Query) Limit() int { return q.PageSize }

// Of 构造分页结果。
func Of[T any](list []T, total int64, q *Query) Result[T] {
	return Result[T]{List: list, Total: total, Page: q.Page, PageSize: q.PageSize}
}
