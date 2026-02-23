package task

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"Kairo/internal/config"
	"Kairo/internal/deps"
	"Kairo/internal/models"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Manager struct {
	ctx            context.Context
	deps           *deps.Manager
	cancelFuncs    map[string]context.CancelFunc
	deletedTasks   map[string]struct{}
	mu             sync.Mutex
	db             *sql.DB
	OnTaskComplete func(task *models.DownloadTask)
	OnTaskFailed   func(task *models.DownloadTask)
}

func NewManager(ctx context.Context, db *sql.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:          ctx,
		deps:         d,
		cancelFuncs:  make(map[string]context.CancelFunc),
		deletedTasks: make(map[string]struct{}),
	}
	m.db = db
	m.resetInterruptedTasks()
	return m
}

func (m *Manager) GetTasks() map[string]*models.DownloadTask {
	tasks := make(map[string]*models.DownloadTask)
	if m.db == nil {
		return tasks
	}

	rows, err := m.db.Query(fmt.Sprintf(`SELECT %s FROM tasks`, taskColumns))
	if err != nil {
		fmt.Printf("Failed to query tasks: %v\n", err)
		return tasks
	}
	defer rows.Close()

	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			continue
		}
		tasks[t.ID] = t
	}
	return tasks
}

func (m *Manager) AddPlaylistTask(input models.AddPlaylistTaskInput) (string, error) {
	dir, err := validateTaskInput(input.URL, input.Dir)
	if err != nil {
		return "", err
	}

	// 1. Create parent task
	parentTask := newTask(input.URL, dir, input.Title, input.Thumbnail, models.SourceTypePlaylist)
	parentTask.Status = models.TaskStatusCompleted
	parentTask.Progress = 100
	parentTask.TotalBytes = 0

	m.registerCancel(parentTask.ID)

	// 2. Create child tasks
	tasks := []*models.DownloadTask{
		parentTask,
	}
	for _, item := range input.PlaylistItems {
		childTask := newTask(item.URL, dir, item.Title, item.Thumbnail, models.SourceTypePlaylist)
		childTask.LogPath = config.GetLogPath(childTask.ID)
		childTask.ParentID = parentTask.ID
		childTask.Quality = "best"
		childTask.Format = "original"

		m.registerCancel(childTask.ID)

		tasks = append(tasks, childTask)
	}

	// 3. Save tasks and emit updates
	for _, t := range tasks {
		m.saveTask(t)
		m.emitTaskUpdate(t)
	}

	go m.scheduleTasks()

	return parentTask.ID, nil
}

func (m *Manager) AddRSSTask(input models.AddRSSTaskInput) (string, error) {
	if input.FeedURL == "" {
		return "", fmt.Errorf("feed url is empty")
	}
	if input.ItemURL == "" {
		return "", fmt.Errorf("item url is empty")
	}
	dir, err := validateTaskInput(input.FeedURL, input.Dir)
	if err != nil {
		return "", err
	}

	// 1. Check for existing parent task
	parentTask, _ := m.findTaskBySourceAndURL(models.SourceTypeRSS, input.FeedURL)

	var tasks []*models.DownloadTask

	// 2. Create parent if not exists
	if parentTask == nil {
		parentTask = newTask(input.FeedURL, dir, input.FeedTitle, input.FeedThumbnail, models.SourceTypeRSS)
		parentTask.Status = models.TaskStatusCompleted
		parentTask.Progress = 100
		parentTask.TotalBytes = 0

		m.registerCancel(parentTask.ID)
		tasks = append(tasks, parentTask)
	}

	// 3. Create child task
	childTask := newTask(input.ItemURL, dir, input.ItemTitle, input.ItemThumbnail, models.SourceTypeRSS)
	childTask.LogPath = config.GetLogPath(childTask.ID)
	childTask.ParentID = parentTask.ID
	childTask.Quality = "best"
	childTask.Format = "original"

	m.registerCancel(childTask.ID)
	tasks = append(tasks, childTask)

	// 4. Save tasks and emit updates
	for _, t := range tasks {
		m.saveTask(t)
		m.emitTaskUpdate(t)
	}

	go m.scheduleTasks()

	return childTask.ID, nil
}

func (m *Manager) AddTask(input models.AddTaskInput) (string, error) {
	dir, err := validateTaskInput(input.URL, input.Dir)
	if err != nil {
		return "", err
	}

	task := newTask(input.URL, dir, input.Title, input.Thumbnail, input.SourceType)
	task.Quality = input.Quality
	task.Format = input.Format
	task.FormatID = input.FormatID
	task.TotalBytes = input.TotalBytes
	task.TrimStart = input.TrimStart
	task.TrimEnd = input.TrimEnd
	task.TrimMode = input.TrimMode

	m.registerCancel(task.ID)
	m.saveTask(task)

	// Emit initial state
	m.emitTaskUpdate(task)

	// Start download in background
	go m.scheduleTasks()

	return task.ID, nil
}

func (m *Manager) scheduleTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	downloadingCount, err := m.getRunningTaskCount()
	if err != nil {
		fmt.Printf("Failed to get running task count: %v\n", err)
		return
	}

	maxConcurrent := config.GetMaxConcurrentDownloads()

	if downloadingCount >= maxConcurrent {
		return
	}

	pendingNeeded := maxConcurrent - downloadingCount
	if pendingNeeded <= 0 {
		return
	}

	pendingTasks, err := m.getPendingTasks(pendingNeeded)
	if err != nil {
		fmt.Printf("Failed to get pending tasks: %v\n", err)
		return
	}

	for _, task := range pendingTasks {
		// Update status immediately to prevent duplicate scheduling
		task.Status = models.TaskStatusStarting
		m.saveTask(task)

		ctx, cancel := context.WithCancel(m.ctx)
		m.cancelFuncs[task.ID] = cancel
		go m.processTask(ctx, task)
	}
}

func (m *Manager) DeleteTask(id string, deleteFile bool) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect all IDs to delete including cascaded ones
	idsToDelete := make(map[string]struct{})
	idsToDelete[id] = struct{}{}

	// Check if the task exists in DB
	task, err := m.getTask(id)
	if err != nil {
		return []string{}, err
	}

	// 1. If it's a playlist or RSS, delete all children
	if task.SourceType == models.SourceTypePlaylist || task.SourceType == models.SourceTypeRSS {
		children, err := m.getTasksByParentID(id)
		if err != nil {
			return []string{}, err
		}
		for _, t := range children {
			idsToDelete[t.ID] = struct{}{}
		}
	}

	// 2. If it's a child task, check if it's the last one
	if task.ParentID != "" {
		siblings, _ := m.getTasksByParentID(task.ParentID)
		hasSiblings := false
		for _, t := range siblings {
			if t.ID != id {
				hasSiblings = true
				break
			}
		}
		if !hasSiblings {
			idsToDelete[task.ParentID] = struct{}{}
		}
	}

	// Execute deletion for all collected IDs
	var deletedIDs []string
	for targetID := range idsToDelete {
		m.deleteTaskInternal(targetID, deleteFile)
		deletedIDs = append(deletedIDs, targetID)
	}

	return deletedIDs, nil
}

func (m *Manager) deleteTaskInternal(id string, deleteFile bool) {
	task, _ := m.getTask(id)

	if task != nil {
		if task.Status == models.TaskStatusMerging || task.Status == models.TaskStatusTrimming {
			return
		}

		if deleteFile {
			targetDir := filepath.Dir(task.FilePath)
			go func(dir string) {
				_ = os.RemoveAll(dir)
			}(targetDir)
		}
	}

	m.deletedTasks[id] = struct{}{}

	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	m.deleteTaskLog(id)

	// 3. Save
	m.db.Exec("DELETE FROM tasks WHERE id = ?", id)
}

func (m *Manager) PauseTask(id string) error {
	m.mu.Lock()
	// Check if running via cancelFuncs
	_, isRunning := m.cancelFuncs[id]
	if !isRunning {
		m.mu.Unlock()
		return nil
	}

	// Load task to update status
	task, err := m.getTask(id)
	if err != nil {
		m.mu.Unlock()
		return err
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
	return nil
}

func (m *Manager) ResumeTask(id string) error {
	task, err := m.getTask(id)
	if err != nil {
		return err
	}

	// Allow resuming from paused or error
	if task.Status != models.TaskStatusPaused && task.Status != models.TaskStatusError {
		return fmt.Errorf("task is not paused or in error state")
	}

	task.Status = models.TaskStatusPending

	m.emitTaskUpdate(task)
	m.saveTask(task)

	ctx, cancel := context.WithCancel(m.ctx)
	m.mu.Lock()
	m.cancelFuncs[id] = cancel
	m.mu.Unlock()

	go m.processTask(ctx, task)
	return nil
}

func (m *Manager) RetryTask(id string) error {
	task, err := m.getTask(id)
	if err != nil {
		return err
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
	return nil
}

func (m *Manager) OpenTaskDir(id string) error {
	task, err := m.getTask(id)
	if err != nil {
		return err
	}

	// Try to open the title directory first, fallback to task.Dir
	targetDir := filepath.Dir(task.FilePath)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return err
	}

	// Open directory
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", targetDir)
	case "darwin":
		cmd = exec.Command("open", targetDir)
	case "linux":
		cmd = exec.Command("xdg-open", targetDir)
	default:
		return nil
	}
	_ = cmd.Start()
	return nil
}

func (m *Manager) emitTaskUpdate(task *models.DownloadTask) {
	m.mu.Lock()
	if _, deleted := m.deletedTasks[task.ID]; deleted {
		m.mu.Unlock()
		return
	}
	t := *task
	m.mu.Unlock()
	wailsRuntime.EventsEmit(m.ctx, "task:update", t)
}

func (m *Manager) saveTask(task *models.DownloadTask) {
	if m.db == nil {
		return
	}

	query := `INSERT OR REPLACE INTO tasks (
		id, url, dir, quality, format, format_id, parent_id, source_type,
		status, progress, title, thumbnail, speed, eta,
		log_path, file_exists, file_path,
		total_bytes, trim_start, trim_end, trim_mode, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.Exec(query,
		task.ID, task.URL, task.Dir, task.Quality, task.Format, task.FormatID, task.ParentID, task.SourceType,
		task.Status, task.Progress, task.Title, task.Thumbnail, task.Speed, task.Eta,
		task.LogPath, task.FileExists, task.FilePath,
		task.TotalBytes, task.TrimStart, task.TrimEnd, task.TrimMode, task.CreatedAt,
	)

	if err != nil {
		fmt.Printf("Failed to save task %s: %v\n", task.ID, err)
	}
}

func (m *Manager) resetInterruptedTasks() {
	if m.db == nil {
		return
	}

	query := `UPDATE tasks SET status = ? WHERE status IN (?, ?, ?, ?)`
	_, err := m.db.Exec(query,
		models.TaskStatusPaused,
		models.TaskStatusDownloading,
		models.TaskStatusStarting,
		models.TaskStatusMerging,
		models.TaskStatusTrimming,
	)
	if err != nil {
		fmt.Printf("Failed to reset interrupted tasks: %v\n", err)
	}
}

func (m *Manager) getTask(id string) (*models.DownloadTask, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	row := m.db.QueryRow(fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ?`, taskColumns), id)
	t, err := scanTask(row)
	return t, err
}

func (m *Manager) getTasksByParentID(parentID string) ([]*models.DownloadTask, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := m.db.Query(fmt.Sprintf(`SELECT %s FROM tasks WHERE parent_id = ?`, taskColumns), parentID)
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

func (m *Manager) findTaskBySourceAndURL(sourceType models.SourceType, url string) (*models.DownloadTask, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	row := m.db.QueryRow(fmt.Sprintf(`SELECT %s FROM tasks WHERE source_type = ? AND url = ? LIMIT 1`, taskColumns), sourceType, url)
	t, err := scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (m *Manager) getRunningTaskCount() (int, error) {
	if m.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var count int
	query := `SELECT count(*) FROM tasks WHERE status IN (?, ?, ?, ?)`
	err := m.db.QueryRow(query,
		models.TaskStatusDownloading,
		models.TaskStatusStarting,
		models.TaskStatusMerging,
		models.TaskStatusTrimming,
	).Scan(&count)

	return count, err
}

func (m *Manager) registerCancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelFuncs[id] = func() {}
}
