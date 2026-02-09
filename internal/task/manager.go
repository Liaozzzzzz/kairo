package task

import (
	"context"
	"fmt"
	"os/exec"
	stdruntime "runtime"
	"sync"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/downloader"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Manager struct {
	ctx           context.Context
	downloader    *downloader.Downloader
	assetProvider downloader.AssetProvider
	tasks         map[string]*models.DownloadTask
	cancelFuncs   map[string]context.CancelFunc
	mu            sync.Mutex
}

func NewManager(ctx context.Context, d *downloader.Downloader, ap downloader.AssetProvider) *Manager {
	m := &Manager{
		ctx:           ctx,
		downloader:    d,
		assetProvider: ap,
		tasks:         make(map[string]*models.DownloadTask),
		cancelFuncs:   make(map[string]context.CancelFunc),
	}
	m.loadTasks()
	return m
}

func (m *Manager) GetTasks() map[string]*models.DownloadTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks
}

func (m *Manager) AddTask(input models.AddTaskInput) (string, error) {
	if input.URL == "" {
		return "", fmt.Errorf("地址为空")
	}
	if input.Dir == "" {
		d, err := config.GetDefaultDownloadDir()
		if err != nil {
			return "", fmt.Errorf("无法获取默认目录")
		}
		input.Dir = d
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	task := &models.DownloadTask{
		ID:          id,
		URL:         input.URL,
		Dir:         input.Dir,
		Quality:     input.Quality,
		Format:      input.Format,
		FormatID:    input.FormatID,
		Status:      models.TaskStatusPending,
		Progress:    0,
		Title:       input.Title,
		Thumbnail:   input.Thumbnail,
		TotalBytes:  input.TotalBytes,
		TotalSize:   utils.FormatBytes(input.TotalBytes),
		CurrentItem: 1,
		TotalItems:  1,
		LogPath:     config.GetLogPath(id),
	}

	if task.Title == "" {
		task.Title = input.URL
	}

	m.mu.Lock()
	m.tasks[id] = task
	m.cancelFuncs[id] = func() {}
	m.mu.Unlock()

	m.saveTasks()

	// Emit initial state
	m.emitTaskUpdate(task)

	// Start download in background
	go m.scheduleTasks()

	return id, nil
}

func (m *Manager) scheduleTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	downloadingCount := 0
	for _, task := range m.tasks {
		if task.Status == models.TaskStatusDownloading || task.Status == models.TaskStatusStarting || task.Status == models.TaskStatusMerging {
			downloadingCount++
		}
	}

	maxConcurrent := config.GetMaxConcurrentDownloads()

	for _, task := range m.tasks {
		if downloadingCount >= maxConcurrent {
			break
		}
		if task.Status == models.TaskStatusPending {
			downloadingCount++
			ctx, cancel := context.WithCancel(m.ctx)
			m.cancelFuncs[task.ID] = cancel
			go m.processTask(ctx, task)
		}
	}
}

func (m *Manager) DeleteTask(id string, deleteFile bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Cancel running task
	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	// 2. Delete file if requested and remove from tasks
	if task, ok := m.tasks[id]; ok {
		if deleteFile && task.FilePath != "" {
			go utils.DeleteFile(task.FilePath)
		}
		delete(m.tasks, id)
	}

	// 3. Save
	m.saveTasksInternal()
}

func (m *Manager) PauseTask(id string) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return
	}

	// If not running, can't pause
	if task.Status != models.TaskStatusDownloading && task.Status != models.TaskStatusStarting && task.Status != models.TaskStatusMerging {
		m.mu.Unlock()
		return
	}

	// Cancel running task
	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	task.Status = models.TaskStatusPaused
	m.mu.Unlock()

	m.emitTaskUpdate(task)
	m.saveTasks()
}

func (m *Manager) ResumeTask(id string) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	m.mu.Unlock()

	if !ok {
		return
	}

	// Allow resuming from paused or error
	if task.Status != models.TaskStatusPaused && task.Status != models.TaskStatusError {
		return
	}

	task.Status = models.TaskStatusPending
	// We don't reset progress like RetryTask

	m.emitTaskUpdate(task)
	m.saveTasks()

	ctx, cancel := context.WithCancel(m.ctx)
	m.mu.Lock()
	m.cancelFuncs[id] = cancel
	m.mu.Unlock()

	go m.processTask(ctx, task)
}

func (m *Manager) RetryTask(id string) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	m.mu.Unlock()

	if !ok {
		return
	}

	task.Status = models.TaskStatusPending
	task.Progress = 0

	m.emitTaskUpdate(task)
	m.saveTasks()

	ctx, cancel := context.WithCancel(m.ctx)
	m.mu.Lock()
	m.cancelFuncs[id] = cancel
	m.mu.Unlock()

	go m.processTask(ctx, task)
}

func (m *Manager) OpenTaskDir(id string) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	m.mu.Unlock()

	if !ok {
		return
	}

	// Open directory
	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", task.Dir)
	case "darwin":
		cmd = exec.Command("open", task.Dir)
	case "linux":
		cmd = exec.Command("xdg-open", task.Dir)
	default:
		return
	}
	_ = cmd.Start()
}

func (m *Manager) emitTaskUpdate(task *models.DownloadTask) {
	m.mu.Lock()
	t := *task
	m.mu.Unlock()
	runtime.EventsEmit(m.ctx, "task:update", t)
}

func (m *Manager) emitTaskLog(id, message string, replace bool) {
	if !replace {
		m.appendTextLog(id, message)
	}
	runtime.EventsEmit(m.ctx, "task:log", map[string]interface{}{
		"id":      id,
		"message": message,
		"replace": replace,
	})
}
