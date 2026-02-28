package category

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"Kairo/internal/models"

	"github.com/google/uuid"
)

type Manager struct {
	ctx context.Context
	db  *sql.DB
}

func NewManager(ctx context.Context, db *sql.DB) *Manager {
	m := &Manager{
		ctx: ctx,
		db:  db,
	}
	return m
}

func (m *Manager) GetCategories() ([]models.Category, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.db.Query(`SELECT id, name, prompt, source, created_at, updated_at FROM categories ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Prompt, &c.Source, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (m *Manager) CreateCategory(name, prompt string) (*models.Category, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	now := time.Now().Unix()
	category := &models.Category{
		ID:        uuid.New().String(),
		Name:      name,
		Prompt:    strings.TrimSpace(prompt),
		Source:    models.CategorySourceCustom,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := m.db.Exec(`INSERT INTO categories (id, name, prompt, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		category.ID, category.Name, category.Prompt, category.Source, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (m *Manager) UpdateCategory(id, name, prompt string) (*models.Category, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	now := time.Now().Unix()
	_, err := m.db.Exec(`UPDATE categories SET name = ?, prompt = ?, updated_at = ? WHERE id = ?`,
		name, strings.TrimSpace(prompt), now, id)
	if err != nil {
		return nil, err
	}

	var category models.Category
	var source sql.NullString
	err = m.db.QueryRow(`SELECT id, name, prompt, source, created_at, updated_at FROM categories WHERE id = ?`, id).
		Scan(&category.ID, &category.Name, &category.Prompt, &source, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, err
	}
	sourceValue := strings.TrimSpace(source.String)
	if sourceValue == "" {
		sourceValue = string(models.CategorySourceCustom)
	}
	category.Source = models.CategorySource(sourceValue)
	return &category, nil
}

func (m *Manager) DeleteCategory(id string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	var source sql.NullString
	err := m.db.QueryRow(`SELECT source FROM categories WHERE id = ?`, id).Scan(&source)
	if err != nil {
		return err
	}
	sourceValue := strings.TrimSpace(source.String)
	if sourceValue == string(models.CategorySourceBuiltin) {
		return fmt.Errorf("category is builtin")
	}
	_, err = m.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}
