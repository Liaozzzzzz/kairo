package dal

import (
	"context"
	"time"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type SubtitleDAL struct {
	db *gorm.DB
}

func NewSubtitleDAL(db *gorm.DB) *SubtitleDAL {
	return &SubtitleDAL{db: db}
}

func (d *SubtitleDAL) ListByVideoID(ctx context.Context, videoID string) ([]schema.VideoSubtitle, error) {
	var subs []schema.VideoSubtitle
	err := d.db.WithContext(ctx).Where("video_id = ?", videoID).Order("created_at asc").Find(&subs).Error
	return subs, err
}

func (d *SubtitleDAL) GetByID(ctx context.Context, id string) (*schema.VideoSubtitle, error) {
	var sub schema.VideoSubtitle
	err := d.db.WithContext(ctx).First(&sub, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (d *SubtitleDAL) ExistsByVideoAndPath(ctx context.Context, videoID, filePath string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&schema.VideoSubtitle{}).Where("video_id = ? AND file_path = ?", videoID, filePath).Count(&count).Error
	return count > 0, err
}

func (d *SubtitleDAL) Create(ctx context.Context, sub *schema.VideoSubtitle) error {
	return d.db.WithContext(ctx).Create(sub).Error
}

func (d *SubtitleDAL) Update(ctx context.Context, sub *schema.VideoSubtitle) error {
	return d.db.WithContext(ctx).Save(sub).Error
}

func (d *SubtitleDAL) DeleteByID(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.VideoSubtitle{}, "id = ?", id).Error
}

func (d *SubtitleDAL) DeleteByVideoID(ctx context.Context, videoID string) error {
	return d.db.WithContext(ctx).Delete(&schema.VideoSubtitle{}, "video_id = ?", videoID).Error
}

func (d *SubtitleDAL) UpdateStatus(ctx context.Context, id string, status int) error {
	return d.db.WithContext(ctx).Model(&schema.VideoSubtitle{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UnixMilli(),
	}).Error
}

func (d *SubtitleDAL) ListByStatus(ctx context.Context, status int) ([]schema.VideoSubtitle, error) {
	var subs []schema.VideoSubtitle
	err := d.db.WithContext(ctx).Where("status = ?", status).Find(&subs).Error
	return subs, err
}

func (d *SubtitleDAL) ListByVideoAndStatus(ctx context.Context, videoID string, status int) ([]schema.VideoSubtitle, error) {
	var subs []schema.VideoSubtitle
	err := d.db.WithContext(ctx).Where("video_id = ? AND status = ?", videoID, status).Find(&subs).Error
	return subs, err
}

func (d *SubtitleDAL) UpdateStatusByStatus(ctx context.Context, fromStatus int, toStatus int) error {
	return d.db.WithContext(ctx).Model(&schema.VideoSubtitle{}).Where("status = ?", fromStatus).Updates(map[string]interface{}{
		"status":     toStatus,
		"updated_at": time.Now().UnixMilli(),
	}).Error
}
