package dal

import (
	"context"
	"time"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type PublishAccountDAL struct {
	db *gorm.DB
}

func NewPublishAccountDAL(db *gorm.DB) *PublishAccountDAL {
	return &PublishAccountDAL{db: db}
}

func (d *PublishAccountDAL) ListAccounts(ctx context.Context, platformID string) ([]schema.PublishAccount, error) {
	db := d.db.WithContext(ctx).Model(&schema.PublishAccount{}).Preload("Platform")
	if platformID != "" && platformID != "all" {
		db = db.Where("platform_id = ?", platformID)
	}
	var accounts []schema.PublishAccount
	err := db.Order("created_at desc").Find(&accounts).Error
	return accounts, err
}

func (d *PublishAccountDAL) SaveAccount(ctx context.Context, account *schema.PublishAccount) error {
	return d.db.WithContext(ctx).Save(account).Error
}

func (d *PublishAccountDAL) GetAccountById(ctx context.Context, id string) (*schema.PublishAccount, error) {
	var account schema.PublishAccount
	err := d.db.WithContext(ctx).
		Preload("Platform").
		First(&account, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (d *PublishAccountDAL) DeleteAccount(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.PublishAccount{}, "id = ?", id).Error
}

func (d *PublishAccountDAL) UpdateAccountStatus(ctx context.Context, id string, status schema.PublishAccountStatus, lastChecked int64) error {
	return d.db.WithContext(ctx).Model(&schema.PublishAccount{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       status,
		"last_checked": lastChecked,
		"updated_at":   time.Now().UnixMilli(),
	}).Error
}
