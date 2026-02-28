package dal

import (
	"context"

	"Kairo/internal/db/schema"

	"gorm.io/gorm"
)

type CategoryDAL struct {
	db *gorm.DB
}

func NewCategoryDAL(db *gorm.DB) *CategoryDAL {
	return &CategoryDAL{db: db}
}

func (d *CategoryDAL) List(ctx context.Context) ([]schema.Category, error) {
	var categories []schema.Category
	err := d.db.WithContext(ctx).Order("created_at asc").Find(&categories).Error
	return categories, err
}

func (d *CategoryDAL) GetByID(ctx context.Context, id string) (*schema.Category, error) {
	var category schema.Category
	err := d.db.WithContext(ctx).First(&category, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (d *CategoryDAL) GetByName(ctx context.Context, name string) (*schema.Category, error) {
	var category schema.Category
	err := d.db.WithContext(ctx).First(&category, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (d *CategoryDAL) Create(ctx context.Context, category *schema.Category) error {
	return d.db.WithContext(ctx).Create(category).Error
}

func (d *CategoryDAL) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return d.db.WithContext(ctx).Model(&schema.Category{}).Where("id = ?", id).Updates(updates).Error
}

func (d *CategoryDAL) Delete(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&schema.Category{}, "id = ?", id).Error
}
