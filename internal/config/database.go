package config

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

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
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
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
			total_size TEXT,
			speed TEXT,
			eta TEXT,
			current_item INTEGER,
			total_items INTEGER,
			log_path TEXT,
			file_exists INTEGER,
			file_path TEXT,
			total_bytes INTEGER,
			trim_start TEXT,
			trim_end TEXT,
			trim_mode TEXT,
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
			subtitles TEXT,
			summary TEXT,
			tags TEXT,
			evaluation TEXT,
			status TEXT
		);
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
			enabled INTEGER DEFAULT 1
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
}
