package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"Kairo/internal/config"
	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"
	"Kairo/internal/deps"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type Manager struct {
	ctx            context.Context
	deps           *deps.Manager
	cancelFuncs    map[string]context.CancelFunc
	deletedTasks   map[string]struct{}
	mu             sync.Mutex
	db             *gorm.DB
	taskDAL        *dal.TaskDAL
	OnTaskComplete func(task *schema.Task)
	OnTaskFailed   func(task *schema.Task)
}

func NewManager(ctx context.Context, db *gorm.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:          ctx,
		deps:         d,
		cancelFuncs:  make(map[string]context.CancelFunc),
		deletedTasks: make(map[string]struct{}),
	}
	m.db = db
	if db != nil {
		m.taskDAL = dal.NewTaskDAL(db)
	}
	m.resetInterruptedTasks()
	return m
}

func (m *Manager) GetTasks() map[string]*schema.Task {
	tasks := make(map[string]*schema.Task)
	if m.taskDAL == nil {
		return tasks
	}

	rows, err := m.taskDAL.List(m.ctx)
	if err != nil {
		fmt.Printf("Failed to query tasks: %v\n", err)
		return tasks
	}
	for _, t := range rows {
		// Create a copy to store in map
		task := t
		tasks[task.ID] = &task
	}
	return tasks
}

func (m *Manager) AddPlaylistTask(input schema.AddPlaylistTaskInput) (string, error) {
	dir, err := validateTaskInput(input.URL, input.Dir)
	if err != nil {
		return "", err
	}

	// 1. Create parent task
	parentTask := newTask(input.URL, dir, input.Title, input.Thumbnail, schema.SourceTypePlaylist, input.CategoryID)
	parentTask.Status = schema.TaskStatusCompleted
	parentTask.Progress = 100
	parentTask.TotalBytes = 0

	m.registerCancel(parentTask.ID)

	// 2. Create child tasks
	tasks := []*schema.Task{
		parentTask,
	}
	for _, item := range input.PlaylistItems {
		childTask := newTask(item.URL, dir, item.Title, item.Thumbnail, schema.SourceTypePlaylist, input.CategoryID)
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

func (m *Manager) AddRSSTask(input schema.AddRSSTaskInput) (string, error) {
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
	parentTask, _ := m.findTaskBySourceAndURL(schema.SourceTypeRSS, input.FeedURL)

	var tasks []*schema.Task

	// 2. Create parent if not exists
	if parentTask == nil {
		parentTask = newTask(input.FeedURL, dir, input.FeedTitle, input.FeedThumbnail, schema.SourceTypeRSS, input.CategoryID)
		parentTask.Status = schema.TaskStatusCompleted
		parentTask.Progress = 100
		parentTask.TotalBytes = 0

		m.registerCancel(parentTask.ID)
		tasks = append(tasks, parentTask)
	}

	// 3. Create child task
	childTask := newTask(input.ItemURL, dir, input.ItemTitle, input.ItemThumbnail, schema.SourceTypeRSS, input.CategoryID)
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

func (m *Manager) AddTask(input schema.AddTaskInput) (string, error) {
	dir, err := validateTaskInput(input.URL, input.Dir)
	if err != nil {
		return "", err
	}

	task := newTask(input.URL, dir, input.Title, input.Thumbnail, input.SourceType, input.CategoryID)
	task.Quality = input.Quality
	task.Format = input.Format
	task.FormatID = input.FormatID
	task.TotalBytes = input.TotalBytes
	task.TrimStart = input.TrimStart
	task.TrimEnd = input.TrimEnd
	task.TrimMode = input.TrimMode
	task.CategoryID = input.CategoryID

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
		task.Status = schema.TaskStatusStarting
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
	if task.SourceType == schema.SourceTypePlaylist || task.SourceType == schema.SourceTypeRSS {
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
		if task.Status == schema.TaskStatusMerging || task.Status == schema.TaskStatusTrimming {
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
	if m.taskDAL != nil {
		_ = m.taskDAL.DeleteByID(m.ctx, id)
	}
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

	task.Status = schema.TaskStatusPaused
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
	if task.Status != schema.TaskStatusPaused && task.Status != schema.TaskStatusError {
		return fmt.Errorf("task is not paused or in error state")
	}

	task.Status = schema.TaskStatusPending

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

	task.Status = schema.TaskStatusPending
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

func (m *Manager) emitTaskUpdate(task *schema.Task) {
	m.mu.Lock()
	if _, deleted := m.deletedTasks[task.ID]; deleted {
		m.mu.Unlock()
		return
	}
	t := *task
	m.mu.Unlock()
	wailsRuntime.EventsEmit(m.ctx, "task:update", t)
}

func (m *Manager) saveTask(task *schema.Task) {
	if m.taskDAL == nil {
		return
	}
	task.UpdatedAt = time.Now().Unix()
	err := m.taskDAL.Save(m.ctx, task)
	if err != nil {
		fmt.Printf("Failed to save task %s: %v\n", task.ID, err)
	}
}

func (m *Manager) resetInterruptedTasks() {
	if m.taskDAL == nil {
		return
	}
	statuses := []string{
		string(schema.TaskStatusDownloading),
		string(schema.TaskStatusStarting),
		string(schema.TaskStatusMerging),
		string(schema.TaskStatusTrimming),
	}
	err := m.taskDAL.ResetInterrupted(m.ctx, string(schema.TaskStatusPaused), statuses)
	if err != nil {
		fmt.Printf("Failed to reset interrupted tasks: %v\n", err)
	}
}

func (m *Manager) getTask(id string) (*schema.Task, error) {
	if m.taskDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return m.taskDAL.GetByID(m.ctx, id)
}

func (m *Manager) getTasksByParentID(parentID string) ([]*schema.Task, error) {
	if m.taskDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.taskDAL.ListByParentID(m.ctx, parentID)
	if err != nil {
		return nil, err
	}
	var tasks []*schema.Task
	for _, t := range rows {
		temp := t
		tasks = append(tasks, &temp)
	}
	return tasks, nil
}

func (m *Manager) findTaskBySourceAndURL(sourceType schema.SourceType, url string) (*schema.Task, error) {
	if m.taskDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	task, err := m.taskDAL.FindBySourceAndURL(m.ctx, int(sourceType), url)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (m *Manager) registerCancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.deletedTasks, id)
}

func (m *Manager) getRunningTaskCount() (int, error) {
	if m.taskDAL == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	statuses := []string{
		string(schema.TaskStatusDownloading),
		string(schema.TaskStatusStarting),
		string(schema.TaskStatusMerging),
		string(schema.TaskStatusTrimming),
	}
	count, err := m.taskDAL.CountByStatus(m.ctx, statuses)
	return int(count), err
}
