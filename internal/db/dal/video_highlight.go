package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type VideoHighlightDAL struct {
	db *gorm.DB
}

func NewVideoHighlightDAL(db *gorm.DB) *VideoHighlightDAL {
	return &VideoHighlightDAL{db: db}
}

func (d *VideoHighlightDAL) ListByVideoID(ctx context.Context, videoID string) ([]schema.VideoHighlight, error) {
	var highlights []schema.VideoHighlight
	db := d.db.WithContext(ctx).Model(&schema.VideoHighlight{})
	if videoID != "" {
		db = db.Where("video_id = ?", videoID)
	}
	err := db.Find(&highlights).Error
	return highlights, err
}

func (d *VideoHighlightDAL) ListByCategoryID(ctx context.Context, categoryID string) ([]schema.VideoHighlight, error) {
	var highlights []schema.VideoHighlight
	// Join with Video table to filter by category_id, but do not preload Video data
	err := d.db.WithContext(ctx).Model(&schema.VideoHighlight{}).
		Joins("JOIN videos ON videos.id = video_highlights.video_id").
		Where("videos.category_id = ?", categoryID).
		Order("videos.created_at desc").
		Find(&highlights).Error
	return highlights, err
}

func (d *VideoHighlightDAL) ListByCategoryIDExcludingPublished(ctx context.Context, categoryID string) ([]schema.VideoHighlight, error) {
	var highlights []schema.VideoHighlight
	err := d.db.WithContext(ctx).
		Table("video_highlights").
		Select("video_highlights.*").
		Joins("JOIN videos ON videos.id = video_highlights.video_id").
		Joins("LEFT JOIN publish_tasks ON publish_tasks.highlight_id = video_highlights.id").
		Where("videos.category_id = ? AND publish_tasks.id IS NULL", categoryID).
		Order("videos.created_at desc").
		Find(&highlights).Error
	return highlights, err
}

func (d *VideoHighlightDAL) GetByID(ctx context.Context, highlightID string) (*schema.VideoHighlight, error) {
	var highlight schema.VideoHighlight
	if err := d.db.WithContext(ctx).First(&highlight, "id = ?", highlightID).Error; err != nil {
		return nil, err
	}
	return &highlight, nil
}

func (d *VideoHighlightDAL) Replace(ctx context.Context, videoID string, highlights []schema.VideoHighlight) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("video_id = ?", videoID).Delete(&schema.VideoHighlight{}).Error; err != nil {
			return err
		}
		if len(highlights) == 0 {
			return nil
		}
		return tx.Create(&highlights).Error
	})
}

func (d *VideoHighlightDAL) UpdateFilePath(ctx context.Context, highlightID, filePath string) error {
	return d.db.WithContext(ctx).Model(&schema.VideoHighlight{}).Where("id = ?", highlightID).Update("file_path", filePath).Error
}

func (d *VideoHighlightDAL) DeleteByVideoID(ctx context.Context, videoID string) error {
	return d.db.WithContext(ctx).Delete(&schema.VideoHighlight{}, "video_id = ?", videoID).Error
}
