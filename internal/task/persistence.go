package task

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/models"
)

func (m *Manager) saveTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveTasksInternal()
}

func (m *Manager) saveTasksInternal() {
	path, err := config.GetStorePath()
	if err != nil {
		return
	}
	data, err := json.MarshalIndent(m.tasks, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0644)
}

func (m *Manager) loadTasks() {
	path, err := config.GetStorePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := json.Unmarshal(data, &m.tasks); err != nil {
		return
	}

	// Reset interrupted tasks
	for _, t := range m.tasks {
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
