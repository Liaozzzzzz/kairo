package task

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (m *Manager) emitTaskLog(id, message string, replace bool) {
	m.mu.Lock()
	if _, deleted := m.deletedTasks[id]; deleted {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	if !replace {
		m.appendTextLog(id, message)
	}
	runtime.EventsEmit(m.ctx, "task:log", map[string]interface{}{
		"id":      id,
		"message": message,
		"replace": replace,
	})
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
