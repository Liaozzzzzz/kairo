package db

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/config"
	"Kairo/internal/db/gormx"
	"Kairo/internal/db/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PublishPlatformSeed struct {
	Name        string
	DisplayName string
	Type        schema.PublishPlatformType
}

func NewDatabase() *gorm.DB {
	settings := config.GetSettings()
	cfg := settings.Database
	if cfg.Type == "" {
		cfg.Type = "sqlite3"
	}
	if cfg.DSN == "" && strings.EqualFold(cfg.Type, "sqlite3") {
		appDir, err := config.GetAppConfigDir()
		if err != nil {
			fmt.Printf("Failed to get app config dir: %v\n", err)
			return nil
		}
		cfg.DSN = filepath.Join(appDir, "tasks.db")
	}

	resolvers := make([]gormx.ResolverConfig, 0, len(cfg.Resolver))
	for _, r := range cfg.Resolver {
		resolvers = append(resolvers, gormx.ResolverConfig{
			DBType:   r.DBType,
			Sources:  r.Sources,
			Replicas: r.Replicas,
			Tables:   r.Tables,
		})
	}

	db, err := gormx.New(gormx.Config{
		Debug:        cfg.Debug,
		PrepareStmt:  cfg.PrepareStmt,
		DBType:       cfg.Type,
		DSN:          cfg.DSN,
		MaxLifetime:  cfg.MaxLifetime,
		MaxIdleTime:  cfg.MaxIdleTime,
		MaxOpenConns: cfg.MaxOpenConns,
		MaxIdleConns: cfg.MaxIdleConns,
		TablePrefix:  cfg.TablePrefix,
		Resolver:     resolvers,
	})
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		return nil
	}

	if strings.EqualFold(cfg.Type, "sqlite3") {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=5000")
		db.Exec("PRAGMA foreign_keys=ON")
		db.Exec("PRAGMA synchronous=NORMAL")
	}

	initSchema(db, cfg.AutoMigrate)
	return db
}

func initSchema(db *gorm.DB, enableAutoMigrate bool) {
	if db == nil {
		return
	}
	if enableAutoMigrate {
		if err := migrateSchema(db); err != nil {
			fmt.Printf("Failed to auto migrate database: %v\n", err)
		}
	}
	seedDefaultCategories(db)
	seedDefaultPublishPlatforms(db)
}

func migrateSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		new(schema.Task),
		new(schema.Video),
		new(schema.VideoSubtitle),
		new(schema.VideoHighlight),
		new(schema.Feed),
		new(schema.FeedItem),
		new(schema.Category),
		new(schema.PublishPlatform),
		new(schema.PublishTask),
		new(schema.PublishAccount),
		new(schema.PublishAutomation),
		new(schema.PublishRecord),
	)
}

func seedDefaultCategories(db *gorm.DB) {
	prompts := ai.GetCategoryPrompts()
	defaults := []struct {
		name   string
		prompt string
	}{
		{
			name:   "英语口语",
			prompt: prompts["英语口语"],
		},
		{
			name:   "演讲",
			prompt: prompts["演讲"],
		},
		{
			name:   "情感关系",
			prompt: prompts["情感关系"],
		},
		{
			name:   "经验分享",
			prompt: prompts["经验分享"],
		},
	}
	now := time.Now().Unix()
	for _, item := range defaults {
		var existing schema.Category
		err := db.Where("name = ?", item.name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		_ = db.Create(&schema.Category{
			ID:        uuid.NewString(),
			Name:      item.name,
			Prompt:    item.prompt,
			Source:    schema.CategorySourceBuiltin,
			CreatedAt: now,
			UpdatedAt: now,
		}).Error
	}
}

func seedDefaultPublishPlatforms(db *gorm.DB) {
	platforms := []PublishPlatformSeed{
		{
			Name:        string(schema.PlatformXiaohongshu),
			DisplayName: "小红书",
			Type:        schema.PublishPlatformTypeBuiltin,
		},
		{
			Name:        string(schema.PlatformDouyin),
			DisplayName: "抖音",
			Type:        schema.PublishPlatformTypeBuiltin,
		},
	}
	for _, p := range platforms {
		var existing schema.PublishPlatform
		err := db.Where("name = ?", p.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		_ = db.Create(&schema.PublishPlatform{
			ID:          uuid.NewString(),
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Type:        p.Type,
		}).Error
	}
}
