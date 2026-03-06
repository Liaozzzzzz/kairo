package dal

import (
	"context"
	"time"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type PublishTaskDAL struct {
	db *gorm.DB
}

func NewPublishTaskDAL(db *gorm.DB) *PublishTaskDAL {
	return &PublishTaskDAL{db: db}
}

func (d *PublishTaskDAL) SaveTask(ctx context.Context, task *schema.PublishTask) error {
	return d.db.WithContext(ctx).Save(task).Error
}

func (d *PublishTaskDAL) GetTaskById(ctx context.Context, id string) (*schema.PublishTask, error) {
	var task schema.PublishTask
	err := d.db.WithContext(ctx).
		Preload("Highlight").
		Preload("Account").
		Preload("Account.Platform").
		First(&task, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *PublishTaskDAL) ExistsByHighlightAndAccount(ctx context.Context, highlightID, accountID string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&schema.PublishTask{}).
		Where("highlight_id = ? AND account_id = ? AND status NOT IN (?, ?)",
			highlightID, accountID, schema.PublishStatusFailed, schema.PublishStatusCancelled).
		Count(&count).Error
	return count > 0, err
}

func (d *PublishTaskDAL) ListAllTasks(ctx context.Context, statusFilter, platformFilter string) ([]schema.PublishTask, error) {
	db := d.db.WithContext(ctx).Model(&schema.PublishTask{}).
		Preload("Highlight").
		Preload("Account").
		Preload("Account.Platform")

	if statusFilter != "" && statusFilter != "all" {
		db = db.Where("publish_tasks.status = ?", statusFilter)
	}
	if platformFilter != "" && platformFilter != "all" {
		db = db.Joins("JOIN publish_accounts ON publish_accounts.id = publish_tasks.account_id").
			Where("publish_accounts.platform_id = ?", platformFilter)
	}

	var tasks []schema.PublishTask
	err := db.Order("publish_tasks.created_at desc").Find(&tasks).Error
	return tasks, err
}

func (d *PublishTaskDAL) ListTasks(ctx context.Context, statusFilter, platformFilter string, page, pageSize int) ([]schema.PublishTask, int64, error) {
	db := d.db.WithContext(ctx).Model(&schema.PublishTask{}).
		Preload("Highlight").
		Preload("Account").
		Preload("Account.Platform")

	if statusFilter != "" && statusFilter != "all" {
		db = db.Where("publish_tasks.status = ?", statusFilter)
	}
	if platformFilter != "" && platformFilter != "all" {
		db = db.Joins("JOIN publish_accounts ON publish_accounts.id = publish_tasks.account_id").
			Where("publish_accounts.platform_id = ?", platformFilter)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []schema.PublishTask
	offset := (page - 1) * pageSize
	err := db.Order("publish_tasks.created_at desc").Offset(offset).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (d *PublishTaskDAL) ListPendingTasks(ctx context.Context) ([]schema.PublishTask, error) {
	var tasks []schema.PublishTask
	nowMs := time.Now().UnixMilli()
	err := d.db.WithContext(ctx).
		Model(&schema.PublishTask{}).
		Where("status = ? AND type = ? AND (scheduled_at = 0 OR scheduled_at <= ?)",
			schema.PublishStatusPending, schema.PublishTypeAuto, nowMs).
		Preload("Highlight").
		Preload("Account").
		Preload("Account.Platform").
		Order("created_at asc").
		Find(&tasks).Error
	return tasks, err
}

func (d *PublishTaskDAL) UpdateTaskStatus(ctx context.Context, id string, status schema.PublishStatus, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    time.Now().UnixMilli(),
	}

	if status == schema.PublishStatusPublished {
		updates["published_at"] = time.Now().UnixMilli()
	}

	return d.db.WithContext(ctx).Model(&schema.PublishTask{}).Where("id = ?", id).Updates(updates).Error
}

func (d *PublishTaskDAL) DeleteTask(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.PublishTask{}, "id = ?", id).Error
}

func (d *PublishTaskDAL) DeleteTasksByVideoID(ctx context.Context, _ string) error {
	return nil
}

func (d *PublishTaskDAL) GetLatestScheduledTask(ctx context.Context, accountID string) (*schema.PublishTask, error) {
	var task schema.PublishTask
	err := d.db.WithContext(ctx).
		Where("account_id = ? AND status = ?", accountID, schema.PublishStatusPending).
		Order("scheduled_at desc").
		Limit(1).
		First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}
