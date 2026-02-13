package task

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	deletedTasks  map[string]struct{}
	mu            sync.Mutex
	db            *sql.DB
}

func NewManager(ctx context.Context, d *downloader.Downloader, ap downloader.AssetProvider) *Manager {
	m := &Manager{
		ctx:           ctx,
		downloader:    d,
		assetProvider: ap,
		tasks:         make(map[string]*models.DownloadTask),
		cancelFuncs:   make(map[string]context.CancelFunc),
		deletedTasks:  make(map[string]struct{}),
	}
	m.initDB()
	m.loadTasks()
	return m
}

func (m *Manager) GetTasks() map[string]*models.DownloadTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks
}

func (m *Manager) AddPlaylistTask(input models.AddPlaylistTaskInput) (string, error) {
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

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Create parent task
	parentID := fmt.Sprintf("%d", time.Now().UnixNano())
	parentTask := &models.DownloadTask{
		ID:            parentID,
		URL:           input.URL,
		Dir:           input.Dir,
		Status:        models.TaskStatusCompleted,
		Progress:      100,
		Title:         input.Title,
		Thumbnail:     input.Thumbnail,
		TotalBytes:    0,
		TotalSize:     utils.FormatBytes(0),
		IsPlaylist:    true,
		PlaylistItems: make([]int, len(input.PlaylistItems)),
		TotalItems:    len(input.PlaylistItems),
		CurrentItem:   len(input.PlaylistItems),
		LogPath:       config.GetLogPath(parentID),
		TrimMode:      models.TrimModeNone,
	}
	if parentTask.Title == "" {
		parentTask.Title = input.URL
	}

	m.tasks[parentID] = parentTask
	m.cancelFuncs[parentID] = func() {}

	// 2. Create child tasks
	for i, item := range input.PlaylistItems {
		childID := fmt.Sprintf("%d_%d", time.Now().UnixNano(), i)
		childTask := &models.DownloadTask{
			ID:          childID,
			URL:         item.URL,
			Dir:         input.Dir,
			Status:      models.TaskStatusPending,
			Progress:    0,
			Title:       item.Title,
			Thumbnail:   item.Thumbnail,
			ParentID:    parentID,
			IsPlaylist:  false,
			Quality:     "best", // Mark as needing best quality
			Format:      "original",
			FormatID:    "", // Will be determined later
			CurrentItem: 1,
			TotalItems:  1,
			LogPath:     config.GetLogPath(childID),
			TrimMode:    models.TrimModeNone,
		}
		if childTask.Title == "" {
			childTask.Title = item.URL
		}
		m.tasks[childID] = childTask
		m.cancelFuncs[childID] = func() {}
		parentTask.PlaylistItems[i] = item.Index // Store index or just ignore
	}

	// 3. Save
	m.saveTask(parentTask)
	for _, t := range m.tasks {
		if t.ParentID == parentID {
			m.saveTask(t)
		}
	}

	// 4. Emit updates
	// Since we are holding the lock, we can't call emitTaskUpdate which locks again.
	// We should probably release lock before emitting or make emitTaskUpdate not lock?
	// But emitTaskUpdate reads from map.
	// Let's iterate and emit after unlocking.
	// But we defer Unlock.
	// So we need to refactor slightly or just schedule a goroutine.
	// Or just saveTasks is enough for persistence, but we need UI update.

	// We can't iterate m.tasks safely after unlock if it changes?
	// But we just added them.
	// Let's create a list of tasks to emit inside the lock.
	var tasksToEmit []models.DownloadTask
	tasksToEmit = append(tasksToEmit, *parentTask)
	for _, t := range m.tasks {
		if t.ParentID == parentID {
			tasksToEmit = append(tasksToEmit, *t)
		}
	}

	go func() {
		for _, t := range tasksToEmit {
			runtime.EventsEmit(m.ctx, "task:update", t)
		}
		m.scheduleTasks()
	}()

	return parentID, nil
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
		ParentID:    "",
		IsPlaylist:  false,
		Status:      models.TaskStatusPending,
		Progress:    0,
		Title:       input.Title,
		Thumbnail:   input.Thumbnail,
		TotalBytes:  input.TotalBytes,
		TotalSize:   utils.FormatBytes(input.TotalBytes),
		CurrentItem: 1,
		TotalItems:  1,
		LogPath:     config.GetLogPath(id),
		TrimStart:   input.TrimStart,
		TrimEnd:     input.TrimEnd,
		TrimMode:    input.TrimMode,
	}

	if task.Title == "" {
		task.Title = input.URL
	}

	m.mu.Lock()
	m.tasks[id] = task
	m.cancelFuncs[id] = func() {}
	m.mu.Unlock()

	m.saveTask(task)

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
		if task.Status == models.TaskStatusDownloading || task.Status == models.TaskStatusStarting || task.Status == models.TaskStatusMerging || task.Status == models.TaskStatusTrimming {
			downloadingCount++
		}
	}

	maxConcurrent := config.GetMaxConcurrentDownloads()

	for _, task := range m.tasks {
		if downloadingCount >= maxConcurrent {
			break
		}
		if task.Status == models.TaskStatusPending && !task.IsPlaylist {
			downloadingCount++
			ctx, cancel := context.WithCancel(m.ctx)
			m.cancelFuncs[task.ID] = cancel
			go m.processTask(ctx, task)
		}
	}
}

func (m *Manager) DeleteTask(id string, deleteFile bool) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect all IDs to delete including cascaded ones
	idsToDelete := make(map[string]struct{})
	idsToDelete[id] = struct{}{}

	// Check if the task exists in memory
	if task, ok := m.tasks[id]; ok {
		// 1. If it's a playlist, delete all children
		if task.IsPlaylist {
			for tid, t := range m.tasks {
				if t.ParentID == id {
					idsToDelete[tid] = struct{}{}
				}
			}
		}

		// 2. If it's a child task, check if it's the last one
		if task.ParentID != "" {
			parentID := task.ParentID
			hasSiblings := false
			for tid, t := range m.tasks {
				if t.ParentID == parentID && tid != id {
					hasSiblings = true
					break
				}
			}
			if !hasSiblings {
				idsToDelete[parentID] = struct{}{}
			}
		}
	}

	// Execute deletion for all collected IDs
	var deletedIDs []string
	for targetID := range idsToDelete {
		m.deleteTaskInternal(targetID, deleteFile)
		deletedIDs = append(deletedIDs, targetID)
	}

	return deletedIDs
}

func (m *Manager) deleteTaskInternal(id string, deleteFile bool) {
	// 1. Cancel running task
	// 2. Delete file if requested and remove from tasks
	if task, ok := m.tasks[id]; ok {
		if task.Status == models.TaskStatusMerging || task.Status == models.TaskStatusTrimming {
			return
		}

		m.deletedTasks[id] = struct{}{}

		if cancel, ok := m.cancelFuncs[id]; ok {
			cancel()
			delete(m.cancelFuncs, id)
		}

		if deleteFile {
			sanitizedTitle := utils.SanitizeFileName(task.Title)
			targetDir := filepath.Join(task.Dir, sanitizedTitle)
			go func(dir string) {
				_ = os.RemoveAll(dir)
			}(targetDir)
		}
		delete(m.tasks, id)
		m.deleteTaskLog(id)
		if task.Status != models.TaskStatusStarting && task.Status != models.TaskStatusDownloading && task.Status != models.TaskStatusMerging && task.Status != models.TaskStatusTrimming {
			delete(m.deletedTasks, id)
		}
	}

	// 3. Save
	m.deleteTaskFromDB(id)
}

func (m *Manager) PauseTask(id string) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return
	}

	// If not running, can't pause
	if task.Status != models.TaskStatusDownloading && task.Status != models.TaskStatusStarting {
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
	m.saveTask(task)
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
	m.saveTask(task)

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
	m.saveTask(task)

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

	// Try to open the title directory first, fallback to task.Dir
	sanitizedTitle := utils.SanitizeFileName(task.Title)
	targetDir := filepath.Join(task.Dir, sanitizedTitle)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		targetDir = task.Dir
	}

	// Open directory
	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", targetDir)
	case "darwin":
		cmd = exec.Command("open", targetDir)
	case "linux":
		cmd = exec.Command("xdg-open", targetDir)
	default:
		return
	}
	_ = cmd.Start()
}

func (m *Manager) emitTaskUpdate(task *models.DownloadTask) {
	m.mu.Lock()
	if _, deleted := m.deletedTasks[task.ID]; deleted {
		m.mu.Unlock()
		return
	}
	t := *task
	m.mu.Unlock()
	runtime.EventsEmit(m.ctx, "task:update", t)
}

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
