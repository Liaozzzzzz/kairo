package dal

import (
	"context"
	"time"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type VideoDAL struct {
	db *gorm.DB
}

func NewVideoDAL(db *gorm.DB) *VideoDAL {
	return &VideoDAL{db: db}
}

func (d *VideoDAL) Save(ctx context.Context, video *schema.Video) error {
	return d.db.WithContext(ctx).Save(video).Error
}

func (d *VideoDAL) ExistsByTaskOrPath(ctx context.Context, taskID, filePath string) (bool, error) {
	query := d.db.WithContext(ctx).Model(&schema.Video{})
	if taskID != "" && filePath != "" {
		query = query.Where("task_id = ? OR file_path = ?", taskID, filePath)
	} else if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	} else {
		query = query.Where("file_path = ?", filePath)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (d *VideoDAL) List(ctx context.Context, statusFilter, query string) ([]schema.Video, error) {
	db := d.db.WithContext(ctx).Model(&schema.Video{})
	if statusFilter != "" && statusFilter != "all" {
		if statusFilter == "analyzed" {
			db = db.Where("status = ?", "completed")
		} else if statusFilter == "unanalyzed" {
			db = db.Where("(status IS NULL OR status = '' OR status = ?)", "none")
		}
	}
	if query != "" {
		db = db.Where("title LIKE ?", "%"+query+"%")
	}
	var videos []schema.Video
	err := db.Order("created_at desc").Find(&videos).Error
	return videos, err
}

func (d *VideoDAL) GetByID(ctx context.Context, id string) (*schema.Video, error) {
	var video schema.Video
	err := d.db.WithContext(ctx).First(&video, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (d *VideoDAL) UpdateStatus(ctx context.Context, id, status, summary, evaluation, tags string) error {
	return d.db.WithContext(ctx).Model(&schema.Video{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"summary":    summary,
		"evaluation": evaluation,
		"tags":       tags,
		"updated_at": time.Now().Unix(),
	}).Error
}

func (d *VideoDAL) ListHighlights(ctx context.Context, videoID string) ([]schema.VideoHighlight, error) {
	var highlights []schema.VideoHighlight
	db := d.db.WithContext(ctx).Model(&schema.VideoHighlight{})
	if videoID != "" {
		db = db.Where("video_id = ?", videoID)
	}
	err := db.Find(&highlights).Error
	return highlights, err
}

func (d *VideoDAL) ListHighlightsByCategoryID(ctx context.Context, categoryID string) ([]schema.VideoHighlight, error) {
	var highlights []schema.VideoHighlight
	// Join with Video table to filter by category_id, but do not preload Video data
	err := d.db.WithContext(ctx).Model(&schema.VideoHighlight{}).
		Joins("JOIN videos ON videos.id = video_highlights.video_id").
		Where("videos.category_id = ?", categoryID).
		Order("videos.created_at desc").
		Find(&highlights).Error
	return highlights, err
}

func (d *VideoDAL) GetHighlightByID(ctx context.Context, highlightID string) (*schema.VideoHighlight, error) {
	var highlight schema.VideoHighlight
	if err := d.db.WithContext(ctx).First(&highlight, "id = ?", highlightID).Error; err != nil {
		return nil, err
	}
	return &highlight, nil
}

func (d *VideoDAL) ReplaceHighlights(ctx context.Context, videoID string, highlights []schema.VideoHighlight) error {
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

func (d *VideoDAL) UpdateHighlightFilePath(ctx context.Context, highlightID, filePath string) error {
	return d.db.WithContext(ctx).Model(&schema.VideoHighlight{}).Where("id = ?", highlightID).Update("file_path", filePath).Error
}

func (d *VideoDAL) DeleteByID(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.Video{}, "id = ?", id).Error
}

func (d *VideoDAL) DeleteHighlightsByVideoID(ctx context.Context, videoID string) error {
	return d.db.WithContext(ctx).Delete(&schema.VideoHighlight{}, "video_id = ?", videoID).Error
}

func (d *VideoDAL) DeleteVideoCascade(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&schema.VideoHighlight{}, "video_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&schema.VideoSubtitle{}, "video_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&schema.Video{}, "id = ?", id).Error
	})
}
