package task

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	_ "github.com/mattn/go-sqlite3"
)

func (m *Manager) initDB() {
	appDir, err := config.GetAppConfigDir()
	if err != nil {
		fmt.Printf("Failed to get app config dir: %v\n", err)
		return
	}
	dbPath := filepath.Join(appDir, "tasks.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		return
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		url TEXT,
		dir TEXT,
		quality TEXT,
		format TEXT,
		format_id TEXT,
		parent_id TEXT,
		is_playlist INTEGER,
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
		files TEXT,
		playlist_items TEXT,
		trim_start TEXT,
		trim_end TEXT,
		trim_mode TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Printf("Failed to create table: %v\n", err)
		return
	}

	m.db = db

	// Try to migrate from JSON if DB is empty
	m.migrateFromJSON()
}

func (m *Manager) migrateFromJSON() {
	jsonPath, err := config.GetStorePath()
	if err != nil {
		return
	}

	// Check if JSON file exists
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return
	}

	// Check if DB is empty
	var count int
	err = m.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	fmt.Println("Migrating tasks from JSON to SQLite...")

	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		fmt.Printf("Failed to read tasks.json: %v\n", err)
		return
	}

	var tasks map[string]*models.DownloadTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		fmt.Printf("Failed to unmarshal tasks.json: %v\n", err)
		return
	}

	// Save to DB
	for _, task := range tasks {
		m.saveTask(task)
	}

	// Rename JSON file to backup
	if err := os.Rename(jsonPath, jsonPath+".bak"); err != nil {
		fmt.Printf("Failed to rename tasks.json: %v\n", err)
	} else {
		fmt.Println("Migration completed, tasks.json renamed to tasks.json.bak")
	}
}

func (m *Manager) saveTask(task *models.DownloadTask) {
	if m.db == nil {
		return
	}

	filesJson, _ := json.Marshal(task.Files)
	playlistItemsJson, _ := json.Marshal(task.PlaylistItems)

	query := `INSERT OR REPLACE INTO tasks (
		id, url, dir, quality, format, format_id, parent_id, is_playlist,
		status, progress, title, thumbnail, total_size, speed, eta,
		current_item, total_items, log_path, file_exists, file_path,
		total_bytes, files, playlist_items, trim_start, trim_end, trim_mode
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.Exec(query,
		task.ID, task.URL, task.Dir, task.Quality, task.Format, task.FormatID, task.ParentID, task.IsPlaylist,
		task.Status, task.Progress, task.Title, task.Thumbnail, task.TotalSize, task.Speed, task.Eta,
		task.CurrentItem, task.TotalItems, task.LogPath, task.FileExists, task.FilePath,
		task.TotalBytes, string(filesJson), string(playlistItemsJson), task.TrimStart, task.TrimEnd, task.TrimMode,
	)

	if err != nil {
		fmt.Printf("Failed to save task %s: %v\n", task.ID, err)
	}
}

func (m *Manager) deleteTaskFromDB(id string) {
	if m.db == nil {
		return
	}
	_, _ = m.db.Exec("DELETE FROM tasks WHERE id = ?", id)
}

func (m *Manager) deleteAllTasksFromDB() {
	if m.db == nil {
		return
	}
	_, _ = m.db.Exec("DELETE FROM tasks")
}

func (m *Manager) loadTasks() {
	if m.db == nil {
		return
	}

	rows, err := m.db.Query("SELECT * FROM tasks")
	if err != nil {
		fmt.Printf("Failed to query tasks: %v\n", err)
		return
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var t models.DownloadTask
		var filesJson string
		var playlistItemsJson string
		var createdAt string // ignore for now

		err := rows.Scan(
			&t.ID, &t.URL, &t.Dir, &t.Quality, &t.Format, &t.FormatID, &t.ParentID, &t.IsPlaylist,
			&t.Status, &t.Progress, &t.Title, &t.Thumbnail, &t.TotalSize, &t.Speed, &t.Eta,
			&t.CurrentItem, &t.TotalItems, &t.LogPath, &t.FileExists, &t.FilePath,
			&t.TotalBytes, &filesJson, &playlistItemsJson, &t.TrimStart, &t.TrimEnd, &t.TrimMode, &createdAt,
		)
		if err != nil {
			continue
		}

		_ = json.Unmarshal([]byte(filesJson), &t.Files)
		_ = json.Unmarshal([]byte(playlistItemsJson), &t.PlaylistItems)

		// Reset interrupted tasks
		if t.Status == models.TaskStatusDownloading || t.Status == models.TaskStatusStarting || t.Status == models.TaskStatusMerging {
			t.Status = models.TaskStatusError
		}

		// Check if file exists
		if t.Status == models.TaskStatusCompleted {
			t.FileExists = false
			if t.FilePath != "" {
				if _, err := os.Stat(t.FilePath); err == nil {
					t.FileExists = true
				}
			}
		}

		if len(t.Files) > 0 {
			for i := range t.Files {
				if t.Files[i].Path != "" {
					t.Files[i].Path = utils.NormalizePath(t.Dir, t.Files[i].Path)
				}
				if info, err := os.Stat(t.Files[i].Path); err == nil {
					t.Files[i].SizeBytes = info.Size()
					t.Files[i].Size = utils.FormatBytes(info.Size())
					if t.Status == models.TaskStatusCompleted {
						t.Files[i].Progress = 100
						t.FileExists = true
					}
				}
			}
		}

		m.tasks[t.ID] = &t
	}
}

func (m *Manager) appendTextLog(id string, message string) {
	path := config.GetLogPath(id)
	if path == "" {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
}

func (m *Manager) deleteTaskLog(id string) {
	path := config.GetLogPath(id)
	if path == "" {
		return
	}
	_ = utils.DeleteFile(path)
}

func (m *Manager) GetTaskLogs(id string) ([]string, error) {
	path := config.GetLogPath(id)
	if path == "" {
		return []string{}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var logs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}

	return logs, scanner.Err()
}
