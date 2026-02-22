package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"Kairo/internal/config"
	"Kairo/internal/deps"
	"Kairo/internal/models"
	"Kairo/internal/rss"
	"Kairo/internal/task"
	"Kairo/internal/utils"
	"Kairo/internal/video"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

// App struct
type App struct {
	ctx          context.Context
	depsManager  *deps.Manager
	taskManager  *task.Manager
	rssManager   *rss.Manager
	videoManager *video.Manager
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

	dep := deps.NewManager(ctx, readEmbedded)
	dep.EnsureYtDlp()
	dep.EnsureFFmpeg()

	a.taskManager = task.NewManager(ctx, dep)
	a.depsManager = dep

	// Initialize Video Manager with DB from Task Manager
	// Ideally DB should be managed by App or a separate module
	a.videoManager = video.NewManager(ctx, a.taskManager.GetDB(), dep)

	// Wire up Task Manager callback to RSS Manager and Video Manager
	a.taskManager.OnTaskComplete = func(task *models.DownloadTask) {
		// Update RSS item status when task completes
		// Note: We use task.URL to match RSS item link
		// This assumes 1-to-1 mapping or at least that the URL is unique enough
		_ = a.rssManager.SetItemDownloadedByLink(task.URL, true)

		// Create video entry from task
		_ = a.videoManager.CreateFromTask(task)
	}

	a.taskManager.OnTaskFailed = func(task *models.DownloadTask) {
		_ = a.rssManager.SetItemFailedByLink(task.URL)
	}

	// Wire up RSS Auto Download
	a.rssManager.OnAutoDownload = func(item models.RSSItem, feed models.RSSFeed) {
		input := models.AddRSSTaskInput{
			FeedURL:       feed.URL,
			FeedTitle:     feed.Title,
			FeedThumbnail: feed.Thumbnail,
			ItemURL:       item.Link,
			ItemTitle:     item.Title,
			Dir:           feed.CustomDir,
		}
		_, err := a.taskManager.AddRSSTask(input)
		if err != nil {
			fmt.Printf("Failed to auto-add RSS task: %v\n", err)
		}
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

// GetVideos returns the list of videos based on filter
func (a *App) GetVideos(filter models.VideoFilter) ([]*models.Video, error) {
	return a.videoManager.GetVideos(filter)
}

// GetVideo returns a single video by ID
func (a *App) GetVideo(id string) (*models.Video, error) {
	return a.videoManager.GetVideo(id)
}

func (a *App) AddVideoToLibrary(taskID string) (bool, error) {
	tasks := a.taskManager.GetTasks()
	task, ok := tasks[taskID]
	if !ok {
		return false, fmt.Errorf("task not found")
	}
	if task.Status != models.TaskStatusCompleted {
		return false, fmt.Errorf("task not completed")
	}
	if !task.FileExists {
		return false, fmt.Errorf("file not found")
	}

	exists, err := a.videoManager.HasVideoForTask(task.ID, task.FilePath)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := a.videoManager.CreateFromTask(task); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteVideo deletes a video from the library
func (a *App) DeleteVideo(id string) error {
	return a.videoManager.DeleteVideo(id)
}

// AnalyzeVideo triggers AI analysis for a video
func (a *App) AnalyzeVideo(id string) error {
	return a.videoManager.AnalyzeVideo(id)
}

// GetVideoHighlights returns highlights for a video
func (a *App) GetVideoHighlights(videoID string) ([]models.AIHighlight, error) {
	return a.videoManager.GetHighlights(videoID)
}

// ClipVideo creates a new video clip and updates the highlight record
func (a *App) ClipVideo(videoID string, highlightID string, start, end string) error {
	v, err := a.videoManager.GetVideo(videoID)
	if err != nil {
		return err
	}

	ffmpegPath, err := a.depsManager.GetFFmpegPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(v.FilePath)
	ext := filepath.Ext(v.FilePath)
	base := strings.TrimSuffix(filepath.Base(v.FilePath), ext)

	safeStart := strings.ReplaceAll(start, ":", "-")
	safeEnd := strings.ReplaceAll(end, ":", "-")
	outputName := fmt.Sprintf("%s_clip_%s_%s%s", base, safeStart, safeEnd, ext)
	outputPath := filepath.Join(dir, outputName)

	args := []string{"-i", v.FilePath, "-ss", start, "-to", end, "-c", "copy", "-y", outputPath}

	cmd := utils.CreateCommand(ffmpegPath, args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}

	// Update the highlight record with the file path
	if err := a.videoManager.UpdateHighlightFilePath(highlightID, outputPath); err != nil {
		return fmt.Errorf("failed to update highlight file path: %v", err)
	}

	// Emit event to update UI
	// We re-emit the full video status which includes highlights
	// Re-fetching the video to get updated highlights
	updatedVideo, err := a.videoManager.GetVideo(videoID)
	if err == nil {
		highlights, _ := a.videoManager.GetHighlights(videoID)
		updatedVideo.Highlights = highlights
		wailsRuntime.EventsEmit(a.ctx, "video:ai_status", map[string]interface{}{
			"id":         updatedVideo.ID,
			"status":     updatedVideo.Status,
			"summary":    updatedVideo.Summary,
			"evaluation": updatedVideo.Evaluation,
			"tags":       updatedVideo.Tags,
			"highlights": updatedVideo.Highlights,
		})
	}

	return nil
}

// FetchSubtitles triggers subtitle download for a video
func (a *App) FetchSubtitles(id string) error {
	return a.videoManager.FetchSubtitles(id)
}

// OpenFile opens a file in the default system application
func (a *App) OpenFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

// ShowInFolder shows a file in the file explorer
func (a *App) ShowInFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,", path)
	case "linux":
		// dbus-send or nautilus depending on environment, falling back to opening dir
		dir := filepath.Dir(path)
		cmd = exec.Command("xdg-open", dir)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
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
	return a.depsManager.GetVideoInfo(url)
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
