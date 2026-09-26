package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/repository"
)

// NotificationService 站内通知服务（000036）。
type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// Broadcast 全员广播。
func (s *NotificationService) Broadcast(ctx context.Context, title, content, level, resource string, resourceID *int64) error {
	return repository.NewNotificationRepo(s.db).Create(ctx, 0, title, content, level, resource, resourceID)
}

// Notify 定向通知到具体用户。
func (s *NotificationService) Notify(ctx context.Context, userID int64, title, content, level, resource string, resourceID *int64) error {
	return repository.NewNotificationRepo(s.db).Create(ctx, userID, title, content, level, resource, resourceID)
}

// List 我的通知（广播 + 定向）。
func (s *NotificationService) List(ctx context.Context, userID int64, onlyUnread bool, page, pageSize int) ([]repository.NotificationRow, int64, error) {
	return repository.NewNotificationRepo(s.db).ListMine(ctx, userID, onlyUnread, (page-1)*pageSize, pageSize)
}

// UnreadCount 我的未读数。
func (s *NotificationService) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	return repository.NewNotificationRepo(s.db).UnreadCount(ctx, userID)
}

// MarkRead 标记已读。
func (s *NotificationService) MarkRead(ctx context.Context, userID, id int64) (bool, error) {
	return repository.NewNotificationRepo(s.db).MarkRead(ctx, userID, id)
}

// MarkAllRead 全部标已读。
func (s *NotificationService) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	return repository.NewNotificationRepo(s.db).MarkAllRead(ctx, userID)
}

// dailySummaryTitle 预警日报广播标题（幂等键）。
const dailySummaryTitle = "库存预警日报"

// NotifyDailyAlertSummary 当日新增未处理预警摘要广播：
// 无新增不打扰；同日已广播则跳过（调度器多任务/重入幂等）。返回是否投递。
func (s *NotificationService) NotifyDailyAlertSummary(ctx context.Context, day time.Time) (bool, error) {
	n, err := repository.NewNotificationRepo(s.db).CountOpenAlertsByDay(ctx, day)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	var exists int64
	if err := s.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id IS NULL AND title = ? AND created_at >= ? AND created_at < ?",
			dailySummaryTitle, start, start.AddDate(0, 0, 1)).
		Count(&exists).Error; err != nil {
		return false, err
	}
	if exists > 0 {
		return false, nil
	}
	msg := fmt.Sprintf("今日新增库存预警 %d 条（效期/下限），请及时处置", n)
	if err := s.Broadcast(ctx, dailySummaryTitle, msg, model.NotifyLevelWarning, "stock_alerts", nil); err != nil {
		return false, err
	}
	return true, nil
}
