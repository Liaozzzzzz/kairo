package main

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"

	"yt-downloader/internal/config"
	"yt-downloader/internal/downloader"
	"yt-downloader/internal/models"
	"yt-downloader/internal/task"

	runtimeapi "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

//go:embed assets/bin/*
var embeddedAssets embed.FS

// readEmbedded reads a file from the embedded assets
func readEmbedded(name string) ([]byte, error) {
	return embeddedAssets.ReadFile("assets/bin/" + name)
}

// App struct
type App struct {
	ctx         context.Context
	downloader  *downloader.Downloader
	taskManager *task.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
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

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.downloader = downloader.NewDownloader(ctx)
	a.taskManager = task.NewManager(ctx, a.downloader, readEmbedded)
	a.downloader.EnsureYtDlp(readEmbedded)
}

func (a *App) ChooseDirectory() (string, error) {
	dir, err := runtimeapi.OpenDirectoryDialog(a.ctx, runtimeapi.OpenDialogOptions{
		Title: "选择下载目录",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
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

func (a *App) GetTasks() map[string]*models.DownloadTask {
	return a.taskManager.GetTasks()
}

func (a *App) DeleteTask(id string) {
	a.taskManager.DeleteTask(id)
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
