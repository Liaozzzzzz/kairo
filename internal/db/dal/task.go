package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type TaskDAL struct {
	db *gorm.DB
}

func NewTaskDAL(db *gorm.DB) *TaskDAL {
	return &TaskDAL{db: db}
}

func (d *TaskDAL) List(ctx context.Context) ([]schema.Task, error) {
	var tasks []schema.Task
	err := d.db.WithContext(ctx).Find(&tasks).Error
	return tasks, err
}

func (d *TaskDAL) GetByID(ctx context.Context, id string) (*schema.Task, error) {
	var task schema.Task
	err := d.db.WithContext(ctx).First(&task, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *TaskDAL) ListByParentID(ctx context.Context, parentID string) ([]schema.Task, error) {
	var tasks []schema.Task
	err := d.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&tasks).Error
	return tasks, err
}

func (d *TaskDAL) FindBySourceAndURL(ctx context.Context, sourceType int, url string) (*schema.Task, error) {
	var task schema.Task
	err := d.db.WithContext(ctx).Where("source_type = ? AND url = ?", sourceType, url).Limit(1).Find(&task).Error
	if err != nil {
		return nil, err
	}
	if task.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &task, nil
}

func (d *TaskDAL) ListPending(ctx context.Context, limit int) ([]schema.Task, error) {
	var tasks []schema.Task
	err := d.db.WithContext(ctx).Where("status = ?", "pending").Order("created_at asc").Limit(limit).Find(&tasks).Error
	return tasks, err
}

func (d *TaskDAL) Save(ctx context.Context, task *schema.Task) error {
	return d.db.WithContext(ctx).Save(task).Error
}

func (d *TaskDAL) DeleteByID(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.Task{}, "id = ?", id).Error
}

func (d *TaskDAL) ResetInterrupted(ctx context.Context, pausedStatus string, statuses []string) error {
	return d.db.WithContext(ctx).Model(&schema.Task{}).Where("status IN ?", statuses).Update("status", pausedStatus).Error
}

func (d *TaskDAL) CountByStatus(ctx context.Context, statuses []string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&schema.Task{}).Where("status IN ?", statuses).Count(&count).Error
	return count, err
}
