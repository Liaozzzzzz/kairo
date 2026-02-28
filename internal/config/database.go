package config

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
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

func InitDatabase() *sql.DB {
	appDir, err := GetAppConfigDir()
	if err != nil {
		fmt.Printf("Failed to get app config dir: %v\n", err)
		return nil
	}
	dbPath := filepath.Join(appDir, "tasks.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		return nil
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(0)
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		fmt.Printf("Failed to set pragma: %v\n", err)
	}
	initSchema(db)
	return db
}

func initSchema(db *sql.DB) {
	if db == nil {
		return
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			url TEXT,
			dir TEXT,
			quality TEXT,
			format TEXT,
			format_id TEXT,
			parent_id TEXT,
			source_type INTEGER,
			status TEXT,
			progress REAL,
			title TEXT,
			thumbnail TEXT,
			speed TEXT,
			eta TEXT,
			log_path TEXT,
			file_exists INTEGER,
			file_path TEXT,
			total_bytes INTEGER,
			trim_start TEXT,
			trim_end TEXT,
			trim_mode TEXT,
			category_id TEXT,
			created_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			task_id TEXT,
			title TEXT,
			url TEXT,
			file_path TEXT,
			thumbnail TEXT,
			duration REAL,
			size INTEGER,
			format TEXT,
			resolution TEXT,
			created_at INTEGER,
			description TEXT,
			uploader TEXT,
			summary TEXT,
			tags TEXT,
			evaluation TEXT,
			category_id TEXT,
			status TEXT
		);
		CREATE TABLE IF NOT EXISTS video_subtitles (
			id TEXT PRIMARY KEY,
			video_id TEXT,
			file_path TEXT,
			language TEXT,
			status INTEGER,
			source INTEGER,
			created_at INTEGER,
			updated_at INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_video_subtitles_video_id ON video_subtitles(video_id);
		CREATE TABLE IF NOT EXISTS video_highlights (
			id TEXT PRIMARY KEY,
			video_id TEXT,
			start_time TEXT,
			end_time TEXT,
			description TEXT,
			file_path TEXT
		);
		CREATE TABLE IF NOT EXISTS feeds (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			thumbnail TEXT,
			last_updated INTEGER,
			unread_count INTEGER DEFAULT 0,
			custom_dir TEXT DEFAULT '',
			download_latest INTEGER DEFAULT 0,
			filters TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			filename_template TEXT DEFAULT '',
			category_id TEXT DEFAULT '',
			enabled INTEGER DEFAULT 1
		);
		CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			prompt TEXT DEFAULT '',
			source TEXT DEFAULT 'custom',
			created_at INTEGER,
			updated_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS feed_items (
			id TEXT PRIMARY KEY,
			feed_id TEXT NOT NULL,
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			description TEXT,
			pub_date INTEGER,
			status INTEGER DEFAULT 0,
			thumbnail TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_items_feed_id ON feed_items(feed_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_items_link ON feed_items(feed_id, link);
	`)
	if err != nil {
		fmt.Printf("Failed to create tables: %v\n", err)
	}

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
	for _, item := range defaults {
		_, _ = db.Exec(
			`INSERT INTO categories (id, name, prompt, source, created_at, updated_at)
			SELECT lower(hex(randomblob(16))), ?, ?, 'builtin', strftime('%s','now'), strftime('%s','now')
			WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name = ? LIMIT 1)`,
			item.name,
			item.prompt,
			item.name,
		)
	}
}
