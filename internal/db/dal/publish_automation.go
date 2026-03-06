package dal

import (
	"context"
	"time"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type PublishAutomationDAL struct {
	db *gorm.DB
}

func NewPublishAutomationDAL(db *gorm.DB) *PublishAutomationDAL {
	return &PublishAutomationDAL{db: db}
}

func (d *PublishAutomationDAL) ListAutomations(ctx context.Context, categoryID string, platformID string) ([]schema.PublishAutomation, error) {
	db := d.db.WithContext(ctx).Model(&schema.PublishAutomation{}).
		Preload("Category").Preload("Account").Preload("Account.Platform")

	if categoryID != "" && categoryID != "all" {
		db = db.Where("category_id = ?", categoryID)
	}

	if platformID != "" && platformID != "all" {
		// Since PlatformID is removed from PublishAutomation, we need to filter by joining with Account
		db = db.Joins("JOIN publish_accounts ON publish_accounts.id = publish_automations.account_id").
			Where("publish_accounts.platform_id = ?", platformID)
	}

	var automations []schema.PublishAutomation
	err := db.Order("publish_automations.created_at desc").Find(&automations).Error
	return automations, err
}

func (d *PublishAutomationDAL) CreateAutomation(ctx context.Context, auto *schema.PublishAutomation) error {
	return d.db.WithContext(ctx).Create(auto).Error
}

func (d *PublishAutomationDAL) GetAutomationById(ctx context.Context, id string) (*schema.PublishAutomation, error) {
	var auto schema.PublishAutomation
	err := d.db.WithContext(ctx).Preload("Category").Preload("Account").Preload("Account.Platform").First(&auto, "id = ?", id).Error
	return &auto, err
}

func (d *PublishAutomationDAL) UpdateAutomation(ctx context.Context, auto *schema.PublishAutomation) error {
	auto.UpdatedAt = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Save(auto).Error
}

func (d *PublishAutomationDAL) DeleteAutomation(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.PublishAutomation{}, "id = ?", id).Error
}

func (d *PublishAutomationDAL) ListEnabledAutomations(ctx context.Context) ([]schema.PublishAutomation, error) {
	var automations []schema.PublishAutomation
	err := d.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		Preload("Category").Preload("Account").Preload("Account.Platform").
		Find(&automations).Error
	return automations, err
}
