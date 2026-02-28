package db

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/db/gormx"
	"Kairo/internal/db/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultAnalysisPrompt = `你是一位顶级的视频内容分析师和短视频策划，请根据输入信息完成内容分析与高能片段识别。

输出要求：
1. 仅输出合法的JSON对象
2. summary、evaluation、highlights、description 使用 {{language}} 语言
3. highlights 按时间先后排序
4. 每个高能片段时长 60-180 秒，避免过短

评估维度：
- 信息价值：信息密度、独特见解
- 情感共鸣：情绪强度、观点鲜明度
- 传播潜力：是否具备传播梗点、金句
- 结构完整性：观点或故事是否有起承转合

你必须输出以下字段：
- "summary": 200字以内的内容总结
- "tags": 5-10个关键词
- "evaluation": 1-3句话的综合评价
- "highlights": 3-6个高能片段，每个包含：
  - "start": HH:MM:SS
  - "end": HH:MM:SS
  - "description": 片段亮点描述，突出冲突/反转/情绪峰值/笑点/金句，避免泛泛而谈

视频信息：
- Title: {{title}}
- Uploader: {{uploader}}
- Date: {{date}}
- Duration: {{duration}}
- Resolution: {{resolution}}
- Format: {{format}}
- Size: {{size}}

内容简介：
{{description}}

字幕统计：
{{subtitle_stats}}

高能候选窗口：
{{energy_candidates}}

字幕节选：
{{subtitles}}`

func InitDatabase() *gorm.DB {
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
	)
}

func seedDefaultCategories(db *gorm.DB) {
	basePrompt := strings.TrimSpace(defaultAnalysisPrompt)
	defaults := []struct {
		name   string
		prompt string
	}{
		{
			name:   "英语口语",
			prompt: basePrompt + "\n\n补充要求：关注口语表达、发音纠错、常用句型、场景化对话、学习方法与练习建议。",
		},
		{
			name:   "演讲",
			prompt: basePrompt + "\n\n补充要求：突出观点结构、开场与收束、论证逻辑、金句与感染力。",
		},
		{
			name:   "情感关系",
			prompt: basePrompt + "\n\n补充要求：强调情感冲突、关系矛盾、观点立场、可执行建议与情绪变化。",
		},
		{
			name:   "经验分享",
			prompt: basePrompt + "\n\n补充要求：突出可执行方法、步骤、避坑经验、适用场景与实践结果。",
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
