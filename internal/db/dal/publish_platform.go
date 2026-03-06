package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type PublishPlatformDAL struct {
	db *gorm.DB
}

func NewPublishPlatformDAL(db *gorm.DB) *PublishPlatformDAL {
	return &PublishPlatformDAL{db: db}
}

func (d *PublishPlatformDAL) ListPlatforms(ctx context.Context) ([]schema.PublishPlatform, error) {
	var platforms []schema.PublishPlatform
	err := d.db.WithContext(ctx).Find(&platforms).Error
	return platforms, err
}

func (d *PublishPlatformDAL) GetPlatformById(ctx context.Context, id string) (*schema.PublishPlatform, error) {
	var platform schema.PublishPlatform
	err := d.db.WithContext(ctx).First(&platform, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

func (d *PublishPlatformDAL) SavePlatform(ctx context.Context, platform *schema.PublishPlatform) error {
	return d.db.WithContext(ctx).Save(platform).Error
}

func (d *PublishPlatformDAL) DeletePlatform(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.PublishPlatform{}, "id = ?", id).Error
}
