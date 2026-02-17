package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"runtime"

	"Kairo/internal/config"
	"Kairo/internal/downloader"
	"Kairo/internal/models"
	"Kairo/internal/rss"
	"Kairo/internal/task"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

// App struct
type App struct {
	ctx         context.Context
	downloader  *downloader.Downloader
	taskManager *task.Manager
	rssManager  *rss.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}
	return a
}

// GetAppVersion returns the current application version
func (a *App) GetAppVersion() string {
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsJSON, &config); err != nil {
		return "unknown"
	}
	return config.Info.ProductVersion
}

// GetPlatform returns the current operating system
func (a *App) GetPlatform() string {
	return runtime.GOOS
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Load settings from disk
	_ = config.LoadSettings()

	a.rssManager = rss.NewManager(ctx)
	a.rssManager.Start()

	d := downloader.NewDownloader(ctx)
	d.EnsureYtDlp(readEmbedded)

	a.taskManager = task.NewManager(ctx, d, readEmbedded)
	a.downloader = d

	// Wire up Task Manager callback to RSS Manager
	a.taskManager.OnTaskComplete = func(task *models.DownloadTask) {
		// Update RSS item status when task completes
		// Note: We use task.URL to match RSS item link
		// This assumes 1-to-1 mapping or at least that the URL is unique enough
		_ = a.rssManager.SetItemDownloadedByLink(task.URL, true)
	}
}

func (a *App) ChooseDirectory() (string, error) {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择下载目录",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) ChooseFile() (string, error) {
	file, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择文件",
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// GetDefaultDownloadDir returns the system download directory
func (a *App) GetDefaultDownloadDir() (string, error) {
	return config.GetDefaultDownloadDir()
}

// GetVideoInfo fetches video metadata
func (a *App) GetVideoInfo(url string) (*models.VideoInfo, error) {
	return a.downloader.GetVideoInfo(url, readEmbedded)
}

// AddTask creates a new download task and starts it
func (a *App) AddTask(input models.AddTaskInput) (string, error) {
	return a.taskManager.AddTask(input)
}

func (a *App) AddPlaylistTask(input models.AddPlaylistTaskInput) (string, error) {
	return a.taskManager.AddPlaylistTask(input)
}

func (a *App) AddRSSTask(input models.AddRSSTaskInput) (string, error) {
	return a.taskManager.AddRSSTask(input)
}

func (a *App) GetTasks() map[string]*models.DownloadTask {
	return a.taskManager.GetTasks()
}

func (a *App) DeleteTask(id string, deleteFile bool) []string {
	return a.taskManager.DeleteTask(id, deleteFile)
}

func (a *App) PauseTask(id string) {
	a.taskManager.PauseTask(id)
}

func (a *App) ResumeTask(id string) {
	a.taskManager.ResumeTask(id)
}

func (a *App) RetryTask(id string) {
	a.taskManager.RetryTask(id)
}

func (a *App) OpenTaskDir(id string) {
	a.taskManager.OpenTaskDir(id)
}

func (a *App) GetTaskLogs(id string) ([]string, error) {
	return a.taskManager.GetTaskLogs(id)
}

// UpdateSettings updates the application settings
func (a *App) UpdateSettings(settings config.AppSettings) {
	config.UpdateSettings(settings)
}

// GetSettings returns the current application settings
func (a *App) GetSettings() config.AppSettings {
	return config.GetSettings()
}

// RSS Methods
func (a *App) AddRSSFeed(input models.AddRSSFeedInput) (*models.RSSFeed, error) {
	return a.rssManager.AddFeed(input)
}

func (a *App) GetRSSFeeds() ([]models.RSSFeed, error) {
	return a.rssManager.GetFeeds()
}

func (a *App) DeleteRSSFeed(id string) error {
	return a.rssManager.DeleteFeed(id)
}

func (a *App) GetRSSFeedItems(feedID string) ([]models.RSSItem, error) {
	return a.rssManager.GetFeedItems(feedID)
}

func (a *App) RefreshRSSFeed(feedID string) error {
	return a.rssManager.RefreshFeed(feedID)
}

func (a *App) MarkRSSItemRead(itemID string) error {
	return a.rssManager.MarkItemRead(itemID)
}

func (a *App) SetRSSFeedEnabled(feedID string, enabled bool) error {
	return a.rssManager.SetFeedEnabled(feedID, enabled)
}

func (a *App) UpdateRSSFeed(feed models.RSSFeed) error {
	return a.rssManager.UpdateFeed(feed)
}

func (a *App) SetRSSItemQueued(itemID string, queued bool) error {
	return a.rssManager.SetItemQueued(itemID, queued)
}
