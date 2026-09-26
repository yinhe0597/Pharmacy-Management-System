package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// NotificationRepo 站内通知仓储（000036，已读按用户隔离存放 notification_reads）。
type NotificationRepo struct{ db *gorm.DB }

func NewNotificationRepo(db *gorm.DB) *NotificationRepo { return &NotificationRepo{db: db} }

// Create 投递通知（userID<=0 时为全员广播）。
func (r *NotificationRepo) Create(ctx context.Context, userID int64, title, content, level, resource string, resourceID *int64) error {
	n := &model.Notification{Title: title, Content: content, Level: level, Resource: resource, ResourceID: resourceID}
	if userID > 0 {
		n.UserID = &userID
	}
	if n.Level == "" {
		n.Level = model.NotifyLevelInfo
	}
	return r.db.WithContext(ctx).Create(n).Error
}

// visibleClause 可见范围：定向给本人 + 全员广播。
const visibleClause = "(n.user_id IS NULL OR n.user_id = ?)"

// NotificationRow 通知列表行（关联回执表拼装已读状态，前端字段口径）。
type NotificationRow struct {
	ID         int64      `json:"id"`
	UserID     *int64     `json:"user_id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Level      string     `json:"level"`
	Resource   string     `json:"resource"`
	ResourceID *int64     `json:"resource_id"`
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ListMine 我的通知（广播 + 定向），按 id 倒序；is_read/read_at 实时关联回执表。
func (r *NotificationRepo) ListMine(ctx context.Context, userID int64, onlyUnread bool, offset, limit int) ([]NotificationRow, int64, error) {
	base := `FROM notifications n LEFT JOIN notification_reads r
	        ON r.notification_id = n.id AND r.user_id = ? WHERE ` + visibleClause
	args := []any{userID, userID}
	if onlyUnread {
		base += " AND r.user_id IS NULL"
	}
	var total int64
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) "+base, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []NotificationRow
	q := `SELECT n.id, n.user_id, n.title, n.content, n.level, n.resource, n.resource_id,
	        (r.user_id IS NOT NULL) AS is_read, r.read_at AS read_at, n.created_at ` + base + ` ORDER BY n.id DESC LIMIT ? OFFSET ?`
	if err := r.db.WithContext(ctx).Raw(q, append(args, limit, offset)...).Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// UnreadCount 我的未读数。
func (r *NotificationRepo) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	var n int64
	q := `SELECT COUNT(*) FROM notifications n LEFT JOIN notification_reads r
	      ON r.notification_id = n.id AND r.user_id = ? WHERE ` + visibleClause + ` AND r.user_id IS NULL`
	if err := r.db.WithContext(ctx).Raw(q, userID, userID).Scan(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// MarkRead 标记单条已读（仅本人可见范围内，返回是否真正更新）。
func (r *NotificationRepo) MarkRead(ctx context.Context, userID, id int64) (bool, error) {
	res := r.db.WithContext(ctx).Exec(`
		INSERT INTO notification_reads (notification_id, user_id, read_at)
		SELECT ?, ?, now()
		WHERE EXISTS (SELECT 1 FROM notifications n WHERE n.id = ? AND `+visibleClause+`)
		ON CONFLICT DO NOTHING`, id, userID, id, userID)
	return res.RowsAffected > 0, res.Error
}

// MarkAllRead 全部标已读（返回新增回执条数）。
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`
		INSERT INTO notification_reads (notification_id, user_id, read_at)
		SELECT n.id, ?, now() FROM notifications n
		LEFT JOIN notification_reads r ON r.notification_id = n.id AND r.user_id = ?
		WHERE `+visibleClause+` AND r.user_id IS NULL
		ON CONFLICT DO NOTHING`, userID, userID, userID)
	return res.RowsAffected, res.Error
}

// CountOpenAlertsByDay 当日新增未处理预警数（调度器摘要口径）。
func (r *NotificationRepo) CountOpenAlertsByDay(ctx context.Context, day time.Time) (int64, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.StockAlert{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "open", start, start.AddDate(0, 0, 1)).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
