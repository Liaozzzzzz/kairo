package category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	ctx context.Context
	db  *gorm.DB
	dal *dal.CategoryDAL
}

func NewManager(ctx context.Context, db *gorm.DB) *Manager {
	m := &Manager{
		ctx: ctx,
		db:  db,
	}
	if db != nil {
		m.dal = dal.NewCategoryDAL(db)
	}
	return m
}

func (m *Manager) GetCategories() ([]schema.Category, error) {
	if m.dal == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.dal.List(m.ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *Manager) CreateCategory(name, prompt string) (*schema.Category, error) {
	if m.dal == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	now := time.Now().Unix()
	category := &schema.Category{
		ID:        uuid.New().String(),
		Name:      name,
		Prompt:    strings.TrimSpace(prompt),
		Source:    schema.CategorySourceCustom,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := m.dal.Create(m.ctx, category)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (m *Manager) UpdateCategory(id, name, prompt string) (*schema.Category, error) {
	if m.dal == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	now := time.Now().Unix()
	err := m.dal.Update(m.ctx, id, map[string]interface{}{
		"name":       name,
		"prompt":     strings.TrimSpace(prompt),
		"updated_at": now,
	})
	if err != nil {
		return nil, err
	}

	updated, err := m.dal.GetByID(m.ctx, id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (m *Manager) DeleteCategory(id string) error {
	if m.dal == nil {
		return fmt.Errorf("database not initialized")
	}
	cat, err := m.dal.GetByID(m.ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if cat.Source == schema.CategorySourceBuiltin {
		return fmt.Errorf("category is builtin")
	}
	return m.dal.Delete(m.ctx, id)
}
