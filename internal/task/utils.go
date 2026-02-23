package task

import (
	"fmt"
	"os"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/models"

	"github.com/google/uuid"
)

const taskColumns = `
	id, url, dir, quality, format, format_id, parent_id, source_type,
	status, progress, title, thumbnail, speed, eta,
	log_path, file_exists, file_path,
	total_bytes, trim_start, trim_end, trim_mode, created_at
`

type Scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s Scanner) (*models.DownloadTask, error) {
	var t models.DownloadTask
	err := s.Scan(
		&t.ID, &t.URL, &t.Dir, &t.Quality, &t.Format, &t.FormatID, &t.ParentID, &t.SourceType,
		&t.Status, &t.Progress, &t.Title, &t.Thumbnail, &t.Speed, &t.Eta,
		&t.LogPath, &t.FileExists, &t.FilePath,
		&t.TotalBytes, &t.TrimStart, &t.TrimEnd, &t.TrimMode, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Check if file exists for completed tasks
	if t.Status == models.TaskStatusCompleted && t.ParentID == "" {
		t.FileExists = false
		if t.FilePath != "" {
			if _, err := os.Stat(t.FilePath); err == nil {
				t.FileExists = true
			}
		}
	}
	return &t, nil
}

func (m *Manager) getPendingTasks(limit int) ([]*models.DownloadTask, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := m.db.Query(fmt.Sprintf(`SELECT %s FROM tasks WHERE status = ? ORDER BY created_at ASC LIMIT ?`, taskColumns), models.TaskStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.DownloadTask
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func validateTaskInput(url, dir string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("地址为空")
	}
	if dir == "" {
		d, err := config.GetDefaultDownloadDir()
		if err != nil {
			return "", fmt.Errorf("无法获取默认目录")
		}
		return d, nil
	}
	return dir, nil
}

func newTask(url, dir, title, thumbnail string, sourceType models.SourceType) *models.DownloadTask {
	id := uuid.New().String()
	return &models.DownloadTask{
		ID:         id,
		URL:        url,
		Dir:        dir,
		Title:      title,
		Thumbnail:  thumbnail,
		SourceType: sourceType,
		Status:     models.TaskStatusPending,
		CreatedAt:  time.Now().Unix(),
		LogPath:    config.GetLogPath(id),
		TrimMode:   models.TrimModeNone,
	}
}
