package task

import (
	"fmt"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/db/schema"

	"github.com/google/uuid"
)

func (m *Manager) getPendingTasks(limit int) ([]*schema.Task, error) {
	if m.taskDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.taskDAL.ListPending(m.ctx, limit)
	if err != nil {
		return nil, err
	}
	var tasks []*schema.Task
	for _, t := range rows {
		// Create a copy to take address
		temp := t
		tasks = append(tasks, &temp)
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

func newTask(url, dir, title, thumbnail string, sourceType schema.SourceType, categoryID string) *schema.Task {
	id := uuid.New().String()
	now := time.Now().Unix()
	return &schema.Task{
		ID:         id,
		URL:        url,
		Dir:        dir,
		Title:      title,
		Thumbnail:  thumbnail,
		SourceType: sourceType,
		Status:     schema.TaskStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		LogPath:    config.GetLogPath(id),
		TrimMode:   schema.TrimModeNone,
		CategoryID: categoryID,
	}
}
