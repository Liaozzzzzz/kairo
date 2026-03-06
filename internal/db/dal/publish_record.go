package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type PublishRecordDAL struct {
	db *gorm.DB
}

func NewPublishRecordDAL(db *gorm.DB) *PublishRecordDAL {
	return &PublishRecordDAL{db: db}
}

func (d *PublishRecordDAL) ListRecords(ctx context.Context, taskID string) ([]schema.PublishRecord, error) {
	var records []schema.PublishRecord
	err := d.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at desc").
		Find(&records).Error
	return records, err
}

func (d *PublishRecordDAL) SaveRecord(ctx context.Context, record *schema.PublishRecord) error {
	return d.db.WithContext(ctx).Save(record).Error
}

func (d *PublishRecordDAL) DeleteRecordsByTaskID(ctx context.Context, taskID string) error {
	return d.db.WithContext(ctx).Delete(&schema.PublishRecord{}, "task_id = ?", taskID).Error
}
