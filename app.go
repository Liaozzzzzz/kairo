package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	runtimeapi "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xi2/xz"
)

//go:embed wails.json
var wailsJSON []byte

// DownloadTask represents a single download task
type DownloadTask struct {
	ID          string       `json:"id"`
	URL         string       `json:"url"`
	Dir         string       `json:"dir"`
	Quality     string       `json:"quality"` // "best", "1080p", "720p", "audio"
	Format      string       `json:"format"`  // "webm", "mp4", "mkv", "avi", "flv", "mov"
	Status      string       `json:"status"`  // "pending", "downloading", "completed", "error"
	Progress    float64      `json:"progress"`
	Title       string       `json:"title"`
	Thumbnail   string       `json:"thumbnail"`
	Stages      []*TaskStage `json:"stages"`
	TotalSize   string       `json:"total_size"`
	Speed       string       `json:"speed"`
	Eta         string       `json:"eta"`
	CurrentItem int          `json:"current_item"`
	TotalItems  int          `json:"total_items"`
	LogPath     string       `json:"log_path"`
	FileExists  bool         `json:"file_exists"`
}

type VideoInfo struct {
	Title     string   `json:"title"`
	Thumbnail string   `json:"thumbnail"`
	Duration  int      `json:"duration"`
	Qualities []string `json:"qualities"`
}

type TaskStage struct {
	Name      string  `json:"name"`       // "Video", "Audio", "Merge"
	Status    string  `json:"status"`     // "pending", "downloading", "completed"
	Progress  float64 `json:"progress"`   // 0-100
	TotalSize string  `json:"total_size"` // Size of this stage
}

// App struct
type App struct {
	ctx         context.Context
	binPath     string
	tasks       map[string]*DownloadTask
	cancelFuncs map[string]context.CancelFunc
	mu          sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		tasks:       make(map[string]*DownloadTask),
		cancelFuncs: make(map[string]context.CancelFunc),
	}
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

func (a *App) getStorePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = home
	}
	appDir := filepath.Join(configDir, "yt-downloader")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "tasks.json"), nil
}

func (a *App) getLogPath(id string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = home
	}
	logDir := filepath.Join(configDir, "yt-downloader", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return ""
	}
	return filepath.Join(logDir, fmt.Sprintf("task_%s.log", id))
}

func (a *App) GetTaskLogs(id string) ([]string, error) {
	path := a.getLogPath(id)
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
	// We might want to limit this if logs are huge, but for now reading all is okay
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}

	return logs, scanner.Err()
}

func (a *App) appendTextLog(id string, message string) {
	path := a.getLogPath(id)
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

func (a *App) saveTasks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.saveTasksInternal()
}

func (a *App) saveTasksInternal() {
	path, err := a.getStorePath()
	if err != nil {
		return
	}
	data, err := json.MarshalIndent(a.tasks, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0644)
}

func (a *App) loadTasks() {
	path, err := a.getStorePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := json.Unmarshal(data, &a.tasks); err != nil {
		return
	}

	// Reset interrupted tasks
	for _, t := range a.tasks {
		if t.Status == "downloading" {
			t.Status = "error"
		}

		// Check if file exists
		if t.Status == "completed" {
			t.FileExists = false
			// Try direct path (if Title contains extension)
			path := filepath.Join(t.Dir, t.Title)
			if _, err := os.Stat(path); err == nil {
				t.FileExists = true
			} else if t.Format != "" {
				// Try appending format (if Title doesn't contain extension)
				pathWithExt := filepath.Join(t.Dir, fmt.Sprintf("%s.%s", t.Title, t.Format))
				if _, err := os.Stat(pathWithExt); err == nil {
					t.FileExists = true
				}
			}
		}
	}
}

func (a *App) GetTasks() map[string]*DownloadTask {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tasks
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadTasks()
	a.ensureYtDlp()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
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
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Downloads"), nil
}

// GetVideoInfo fetches video metadata
func (a *App) GetVideoInfo(url string) (*VideoInfo, error) {
	if url == "" {
		return nil, errors.New("url is empty")
	}

	a.ensureYtDlp()
	if a.binPath == "" {
		return nil, errors.New("yt-dlp not found")
	}

	// Use --dump-json to get metadata
	cmd := exec.Command(a.binPath, "--dump-json", "--no-playlist", url)

	// Set to hide window on Windows if needed (runtime specific),
	// but standard exec usually works fine in Wails background.

	output, err := cmd.Output()
	if err != nil {
		// Try to capture stderr for better error message
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var rawInfo struct {
		Title     string                   `json:"title"`
		Thumbnail string                   `json:"thumbnail"`
		Duration  int                      `json:"duration"`
		Formats   []map[string]interface{} `json:"formats"`
	}

	if err := json.Unmarshal(output, &rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	info := VideoInfo{
		Title:     rawInfo.Title,
		Thumbnail: rawInfo.Thumbnail,
		Duration:  rawInfo.Duration,
	}

	// Parse formats to extract unique heights
	uniqueHeights := make(map[int]bool)
	for _, f := range rawInfo.Formats {
		// Check if it's a video stream
		// vcodec can be "none", or missing.
		isVideo := false
		if vcodec, ok := f["vcodec"].(string); ok {
			if vcodec != "none" {
				isVideo = true
			}
		} else {
			// If vcodec is missing, check if width/height exists and > 0
			if _, ok := f["width"]; ok {
				isVideo = true
			}
		}

		if isVideo {
			if h, ok := f["height"].(float64); ok && h > 0 {
				uniqueHeights[int(h)] = true
			}
		}
	}

	var heights []int
	for h := range uniqueHeights {
		heights = append(heights, h)
	}
	// Sort descending
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))

	var qualities []string
	qualities = append(qualities, "best") // Always include best available

	for _, h := range heights {
		qualities = append(qualities, fmt.Sprintf("%dp", h))
	}
	qualities = append(qualities, "audio") // Always include audio only

	info.Qualities = qualities

	return &info, nil
}

// AddTask creates a new download task and starts it
func (a *App) AddTask(url string, quality string, format string, dir string, title string, thumbnail string) (string, error) {
	if url == "" {
		return "", errors.New("地址为空")
	}
	if dir == "" {
		d, err := a.GetDefaultDownloadDir()
		if err != nil {
			return "", errors.New("无法获取默认目录")
		}
		dir = d
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	task := &DownloadTask{
		ID:        id,
		URL:       url,
		Dir:       dir,
		Quality:   quality,
		Format:    format,
		Status:    "pending",
		Progress:  0,
		Title:     title,
		Thumbnail: thumbnail,
		Stages: []*TaskStage{
			{Name: "video", Status: "pending", Progress: 0},
			{Name: "audio", Status: "pending", Progress: 0},
			{Name: "merge", Status: "pending", Progress: 0},
		},
		CurrentItem: 1,
		TotalItems:  1,
		LogPath:     a.getLogPath(id),
	}

	if task.Title == "" {
		task.Title = url
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.tasks[id] = task
	a.cancelFuncs[id] = cancel
	a.mu.Unlock()

	a.saveTasks()

	// Emit initial state
	a.emitTaskUpdate(task)

	// Start download in background
	go a.processTask(ctx, task)

	return id, nil
}

func (a *App) emitTaskUpdate(task *DownloadTask) {
	a.mu.Lock()
	// Create a copy to avoid race conditions if needed, strictly speaking
	// simplistic copy here
	t := *task
	a.mu.Unlock()
	runtimeapi.EventsEmit(a.ctx, "task:update", t)
}

func (a *App) processTask(ctx context.Context, task *DownloadTask) {
	defer func() {
		a.mu.Lock()
		delete(a.cancelFuncs, task.ID)
		a.mu.Unlock()
	}()

	a.ensureYtDlp()
	if a.binPath == "" {
		task.Status = "error"
		a.emitTaskLog(task.ID, "Error: yt-dlp not found", false)
		a.emitTaskUpdate(task)
		return
	}

	a.mu.Lock()
	task.Status = "downloading"
	task.Speed = "启动中..."
	task.Eta = ""
	// Set the first non-completed stage to downloading
	for _, stage := range task.Stages {
		if stage.Status != "completed" {
			stage.Status = "downloading"
			break
		}
	}
	a.mu.Unlock()

	a.emitTaskLog(task.ID, "正在启动下载引擎...", false)
	a.emitTaskUpdate(task)
	a.saveTasks()

	// Build args based on quality
	format := "bestvideo+bestaudio/best"
	switch task.Quality {
	case "1080p":
		format = "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "720p":
		format = "bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "480p":
		format = "bestvideo[height<=480]+bestaudio/best[height<=480]"
	case "360p":
		format = "bestvideo[height<=360]+bestaudio/best[height<=360]"
	case "240p":
		format = "bestvideo[height<=240]+bestaudio/best[height<=240]"
	case "144p":
		format = "bestvideo[height<=144]+bestaudio/best[height<=144]"
	case "audio":
		format = "bestaudio/best"
	}

	args := []string{
		"--newline", // Important for parsing
		// "--js-runtimes", "node,deno", // Removed: auto-detection is preferred, or use specific path if needed
		"--ffmpeg-location", filepath.Dir(a.binPath), // Explicitly set ffmpeg location
		"-o", "%(title)s.%(ext)s",
		"-P", task.Dir,
		"-f", format,
		"--playlist-items", "1",
	}

	if task.Format != "" {
		args = append(args, "--merge-output-format", task.Format)
	}

	args = append(args, task.URL)

	cmd := exec.CommandContext(ctx, a.binPath, args...)

	// Separate pipes for stdout and stderr
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		a.mu.Lock()
		isPaused := task.Status == "paused"
		a.mu.Unlock()
		if isPaused {
			return
		}

		task.Status = "error"
		a.emitTaskLog(task.ID, "Start Error: "+err.Error(), false)
		a.emitTaskUpdate(task)
		a.saveTasks()
		return
	}

	a.emitTaskLog(task.ID, "下载引擎已启动，正在解析...", false)

	// Regex for progress
	// [download]  25.0% of 10.00MiB at  1.00MiB/s ETA 00:05
	progressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+([~\d\.\w]+)(?:\s+at\s+([~\d\.\w/]+)\s+ETA\s+([\d:]+))?`)
	destinationRegex := regexp.MustCompile(`\[download\] Destination: (.+)`)
	alreadyDownloadedRegex := regexp.MustCompile(`\[download\] (.+) has already been downloaded`)
	mergerRegex := regexp.MustCompile(`\[Merger\] Merging formats`)
	playlistRegex := regexp.MustCompile(`\[download\] Downloading item (\d+) of (\d+)`)

	var wg sync.WaitGroup
	wg.Add(2)

	currentStageIdx := 0
	filesCount := 0

	// Playlist tracking
	currentItem := task.CurrentItem
	if currentItem < 1 {
		currentItem = 1
	}
	totalItems := task.TotalItems
	if totalItems < 1 {
		totalItems = 1
	}
	baseTitle := task.Title

	// Stdout reader
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			isProgress := strings.HasPrefix(line, "[download]") && strings.Contains(line, "%")
			a.emitTaskLog(task.ID, line, isProgress)

			// Parse playlist info
			if matches := playlistRegex.FindStringSubmatch(line); len(matches) > 2 {
				c, _ := strconv.Atoi(matches[1])
				t, _ := strconv.Atoi(matches[2])
				if t > 0 {
					currentItem = c
					totalItems = t
					task.CurrentItem = currentItem
					task.TotalItems = totalItems

					// Reset per-file tracking
					currentStageIdx = 0
					filesCount = 0
					// Reset stages
					for _, s := range task.Stages {
						s.Status = "pending"
						s.Progress = 0
					}
					task.Stages[0].Status = "downloading"

					// Update title prefix
					task.Title = fmt.Sprintf("[%d/%d] %s", currentItem, totalItems, baseTitle)
					a.emitTaskUpdate(task)
				}
			}

			// Parse already downloaded
			if alreadyDownloadedRegex.MatchString(line) {
				// Only if we haven't already marked this stage as complete/downloading
				// This avoids issues where resume might trigger this for an already processed file

				if currentStageIdx < len(task.Stages) && task.Stages[currentStageIdx].Status != "completed" {
					task.Stages[currentStageIdx].Status = "completed"
					task.Stages[currentStageIdx].Progress = 100

					// Move to next stage logic
					if currentStageIdx < len(task.Stages)-1 {
						currentStageIdx++
						task.Stages[currentStageIdx].Status = "downloading"
					}
					filesCount++
					a.emitTaskUpdate(task)
				}
			}

			// Parse stages
			if destinationRegex.MatchString(line) {
				if filesCount > 0 && currentStageIdx == 0 {
					// Moving to second file (Audio)
					task.Stages[0].Status = "completed"
					task.Stages[0].Progress = 100
					currentStageIdx = 1
				}
				if currentStageIdx < len(task.Stages) {
					task.Stages[currentStageIdx].Status = "downloading"
				}
				filesCount++

				parts := strings.Split(line, ": ")
				if len(parts) > 1 {
					filename := filepath.Base(parts[1])
					// Keep the playlist prefix if exists
					if totalItems > 1 {
						task.Title = fmt.Sprintf("[%d/%d] %s", currentItem, totalItems, filename)
					} else {
						if task.Title == task.URL {
							task.Title = filename
							baseTitle = filename // Update base title for single file or first discovery
						}
					}
				}
				a.emitTaskUpdate(task)
			}

			if mergerRegex.MatchString(line) {
				if currentStageIdx < len(task.Stages) {
					task.Stages[currentStageIdx].Status = "completed"
					task.Stages[currentStageIdx].Progress = 100
				}
				currentStageIdx = 2
				if currentStageIdx < len(task.Stages) {
					task.Stages[currentStageIdx].Status = "downloading"
				}
				a.emitTaskUpdate(task)
			}

			// Parse progress
			matches := progressRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				p, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					// Update info
					if len(matches) > 2 && matches[2] != "" && currentStageIdx < len(task.Stages) {
						// 尝试解析大小，例如 "10.00MiB"
						if size, err := parseSize(matches[2]); err == nil {
							// Update current stage size
							task.Stages[currentStageIdx].TotalSize = formatSize(size)

							// Calculate total size by accumulating all stages
							totalBytes := size
							for idx, stage := range task.Stages {
								if idx >= currentStageIdx {
									break
								}
								if s, err := parseSize(stage.TotalSize); err == nil {
									totalBytes += s
								}
							}
							task.TotalSize = formatSize(totalBytes)
						}
					}
					if len(matches) > 3 && matches[3] != "" {
						task.Speed = matches[3]
					}
					if len(matches) > 4 && matches[4] != "" {
						task.Eta = matches[4]
					}

					if currentStageIdx < len(task.Stages) {
						task.Stages[currentStageIdx].Progress = p
					}

					fileProgress := 0.0
					if len(task.Stages) == 3 {
						// Check if we are downloading video or audio
						// Stage 0 is Video, Stage 1 is Audio
						// If we are at Stage 0, we contribute to first 45%
						// If we are at Stage 1, we contribute to next 45% (so 45 + p*0.45)
						// If we are at Stage 2, we contribute to final 10% (so 90 + p*0.1)

						if currentStageIdx == 0 {
							fileProgress = task.Stages[0].Progress * 0.45
						} else if currentStageIdx == 1 {
							fileProgress = 45.0 + (task.Stages[1].Progress * 0.45)
						} else if currentStageIdx == 2 {
							fileProgress = 90.0 + (task.Stages[2].Progress * 0.1)
						}
					} else {
						fileProgress = p
					}

					// Calculate global progress based on playlist
					// Each item contributes (100 / totalItems)
					globalProgress := (float64(currentItem-1)*100.0 + fileProgress) / float64(totalItems)
					task.Progress = globalProgress

					runtimeapi.EventsEmit(a.ctx, "task:progress", map[string]interface{}{
						"id":         task.ID,
						"progress":   globalProgress,
						"total_size": task.TotalSize,
						"speed":      task.Speed,
						"eta":        task.Eta,
						"stages":     task.Stages,
					})
				}
			}
		}
	}()

	// Stderr reader
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			a.emitTaskLog(task.ID, line, false)
		}
	}()

	err := cmd.Wait()
	wg.Wait()

	if err != nil {
		// Check for exit code 101 (Max downloads reached)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 101 {
			task.Status = "completed"
			task.Progress = 100
			for _, s := range task.Stages {
				s.Status = "completed"
				s.Progress = 100
			}
			a.emitTaskLog(task.ID, "Download limit reached (expected)", false)
		} else {
			// Check if paused
			a.mu.Lock()
			isPaused := task.Status == "paused"
			a.mu.Unlock()

			if isPaused {
				return
			}

			task.Status = "error"
			a.emitTaskLog(task.ID, "Exit Error: "+err.Error(), false)
		}
	} else {
		task.Status = "completed"
		task.Progress = 100
		task.FileExists = true
		for _, s := range task.Stages {
			s.Status = "completed"
			s.Progress = 100
		}
	}
	a.emitTaskUpdate(task)
	a.saveTasks()
}

func (a *App) emitTaskLog(id, message string, replace bool) {
	if !replace {
		a.appendTextLog(id, message)
	}
	runtimeapi.EventsEmit(a.ctx, "task:log", map[string]interface{}{
		"id":      id,
		"message": message,
		"replace": replace,
	})
}

func (a *App) ensureYtDlp() {
	if a.binPath != "" {
		if _, err := os.Stat(a.binPath); err == nil {
			return
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		home, _ := os.UserHomeDir()
		cfg = filepath.Join(home, ".config")
	}
	base := filepath.Join(cfg, "yt-downloader", "bin")
	_ = os.MkdirAll(base, 0o755)

	// Ensure ffmpeg
	a.ensureFFmpeg(base)

	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{"yt-dlp_macos", "yt-dlp"}
	case "windows":
		candidates = []string{"yt-dlp.exe"}
	default:
		candidates = []string{"yt-dlp"}
	}
	for _, name := range candidates {
		final := filepath.Join(base, name)
		if _, err := os.Stat(final); err == nil {
			a.binPath = final
			return
		}
		if data, err := readEmbedded(name); err == nil && len(data) > 1_000_000 {
			tmp := final + ".tmp"
			if err := os.WriteFile(tmp, data, 0o755); err == nil {
				_ = os.Chmod(tmp, 0o755)
				if err := os.Rename(tmp, final); err == nil {
					a.binPath = final
					runtimeapi.LogInfo(a.ctx, "已从内置资源安装 yt-dlp")
					return
				}
			}
		}
	}
	for _, name := range candidates {
		url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + name
		runtimeapi.LogInfo(a.ctx, "正在下载 yt-dlp: "+name)
		req, _ := http.NewRequest("GET", url, nil)
		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			runtimeapi.LogError(a.ctx, err.Error())
			continue
		}
		if resp.StatusCode != http.StatusOK {
			runtimeapi.LogError(a.ctx, fmt.Sprintf("下载失败: %s", resp.Status))
			resp.Body.Close()
			continue
		}
		tmp := filepath.Join(base, name+".tmp")
		f, err := os.Create(tmp)
		if err != nil {
			runtimeapi.LogError(a.ctx, err.Error())
			resp.Body.Close()
			continue
		}
		_, err = io.Copy(f, resp.Body)
		resp.Body.Close()
		_ = f.Close()
		if err != nil {
			runtimeapi.LogError(a.ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		final := filepath.Join(base, name)
		_ = os.Chmod(tmp, 0o755)
		if err := os.Rename(tmp, final); err != nil {
			runtimeapi.LogError(a.ctx, err.Error())
			_ = os.Remove(tmp)
			continue
		}
		a.binPath = final
		runtimeapi.LogInfo(a.ctx, "yt-dlp 下载完成")
		return
	}
}

func (a *App) ensureFFmpeg(base string) {
	ffmpegPath := filepath.Join(base, "ffmpeg")
	if runtime.GOOS == "windows" {
		ffmpegPath += ".exe"
	}

	// Check if already exists
	if _, err := os.Stat(ffmpegPath); err == nil {
		return
	}

	// Try to extract from embedded assets first
	embeddedName := "ffmpeg"
	if runtime.GOOS == "windows" {
		embeddedName = "ffmpeg.exe"
	}
	// Note: In a real cross-platform build, we would need to embed binaries for all OSs
	// For this specific environment (macOS), we have downloaded the macOS binary to assets/bin/ffmpeg

	if data, err := readEmbedded(embeddedName); err == nil && len(data) > 0 {
		runtimeapi.LogInfo(a.ctx, "正在从内置资源安装 FFmpeg...")
		tmp := ffmpegPath + ".tmp"
		if err := os.WriteFile(tmp, data, 0o755); err == nil {
			_ = os.Chmod(tmp, 0o755)
			if err := os.Rename(tmp, ffmpegPath); err == nil {
				runtimeapi.LogInfo(a.ctx, "FFmpeg 内置资源安装成功")
				return
			}
		}
	}

	// Try to find in path first
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		// Symlink or copy? Symlink is better
		_ = os.Symlink(path, ffmpegPath)
		return
	}

	runtimeapi.LogInfo(a.ctx, "正在下载 FFmpeg...")

	// Download from yt-dlp/FFmpeg-Builds
	url := ""
	switch runtime.GOOS {
	case "darwin":
		url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-macos64-gpl.zip"
	case "linux":
		url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz"
	case "windows":
		url = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
	}

	if url == "" {
		runtimeapi.LogWarning(a.ctx, "不支持的操作系统，无法自动下载 FFmpeg")
		return
	}

	// Create temp file
	tmpArchive := filepath.Join(base, "ffmpeg_archive.tmp")
	defer os.Remove(tmpArchive)

	// Download
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		runtimeapi.LogError(a.ctx, "FFmpeg 下载请求失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		runtimeapi.LogError(a.ctx, fmt.Sprintf("FFmpeg 下载失败: %s", resp.Status))
		return
	}

	out, err := os.Create(tmpArchive)
	if err != nil {
		runtimeapi.LogError(a.ctx, "无法创建临时文件: "+err.Error())
		return
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		runtimeapi.LogError(a.ctx, "写入文件失败: "+err.Error())
		return
	}

	runtimeapi.LogInfo(a.ctx, "FFmpeg 下载完成，正在解压...")

	// Extract
	var extractErr error
	if strings.HasSuffix(url, ".zip") {
		extractErr = extractZip(tmpArchive, base, "ffmpeg")
	} else {
		extractErr = extractTarXz(tmpArchive, base, "ffmpeg")
	}

	if extractErr != nil {
		runtimeapi.LogError(a.ctx, "FFmpeg 解压失败: "+extractErr.Error())
	} else {
		// Make executable
		_ = os.Chmod(ffmpegPath, 0o755)
		runtimeapi.LogInfo(a.ctx, "FFmpeg 安装成功")
	}
}

func extractZip(src, dest, targetFile string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/"+targetFile) || strings.HasSuffix(f.Name, "\\"+targetFile) || f.Name == targetFile || strings.HasSuffix(f.Name, targetFile+".exe") {
			// Found it
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			finalPath := filepath.Join(dest, filepath.Base(f.Name))
			outFile, err := os.Create(finalPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}
	}
	return errors.New("ffmpeg binary not found in zip")
}

func extractTarXz(src, dest, targetFile string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	// xz.NewReader requires a dict cap. 0 means default.
	xzR, err := xz.NewReader(f, 0)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzR)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasSuffix(header.Name, "/"+targetFile) || header.Name == targetFile {
			finalPath := filepath.Join(dest, filepath.Base(header.Name))
			outFile, err := os.Create(finalPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tr)
			return err
		}
	}
	return errors.New("ffmpeg binary not found in tar.xz")
}

// Deprecated: StartDownload kept for compatibility if needed, but we should use AddTask
func (a *App) StartDownload(url string, dir string) error {
	_, err := a.AddTask(url, "best", "webm", dir, "", "")
	return err
}

func (a *App) DeleteTask(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Cancel running task
	if cancel, ok := a.cancelFuncs[id]; ok {
		cancel()
		delete(a.cancelFuncs, id)
	}

	// 2. Remove from tasks
	delete(a.tasks, id)

	// 3. Save
	a.saveTasksInternal()
}

func (a *App) OpenTaskDir(id string) {
	a.mu.Lock()
	task, ok := a.tasks[id]
	a.mu.Unlock()

	if !ok {
		return
	}

	// Open directory
	var cmd *exec.Cmd
	switch runtime.GOOS {
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

func (a *App) RetryTask(id string) {
	a.mu.Lock()
	task, ok := a.tasks[id]
	a.mu.Unlock()

	if !ok {
		return
	}

	// Only retry if not currently running (simple check)
	// Or force restart
	// If it's running, we should stop it first?
	// Let's assume user clicks retry on error/stopped tasks

	task.Status = "pending"
	task.Progress = 0
	for _, s := range task.Stages {
		s.Status = "pending"
		s.Progress = 0
	}

	a.emitTaskUpdate(task)
	a.saveTasks()

	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.cancelFuncs[id] = cancel
	a.mu.Unlock()

	go a.processTask(ctx, task)
}

func (a *App) PauseTask(id string) {
	a.mu.Lock()
	task, ok := a.tasks[id]
	if !ok {
		a.mu.Unlock()
		return
	}

	// If not downloading, can't pause
	if task.Status != "downloading" {
		a.mu.Unlock()
		return
	}

	// Cancel running task
	if cancel, ok := a.cancelFuncs[id]; ok {
		cancel()
		delete(a.cancelFuncs, id)
	}

	task.Status = "paused"
	a.mu.Unlock()

	a.emitTaskUpdate(task)
	a.saveTasks()
}

func (a *App) ResumeTask(id string) {
	a.mu.Lock()
	task, ok := a.tasks[id]
	a.mu.Unlock()

	if !ok {
		return
	}

	// Allow resuming from paused or error
	if task.Status != "paused" && task.Status != "error" {
		return
	}

	task.Status = "pending"
	// We don't reset progress like RetryTask

	a.emitTaskUpdate(task)
	a.saveTasks()

	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.cancelFuncs[id] = cancel
	a.mu.Unlock()

	go a.processTask(ctx, task)
}

// parseSize parses a size string (e.g. "10.00MiB") into bytes
func parseSize(sizeStr string) (float64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.TrimPrefix(sizeStr, "~")
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Find the split point between number and unit
	var i int
	for i = 0; i < len(sizeStr); i++ {
		if (sizeStr[i] < '0' || sizeStr[i] > '9') && sizeStr[i] != '.' {
			break
		}
	}

	if i == 0 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	numStr := sizeStr[:i]
	unitStr := sizeStr[i:]

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}

	if unitStr == "" {
		return val, nil
	}

	switch unitStr {
	case "B":
		return val, nil
	case "KiB", "K", "k":
		return val * 1024, nil
	case "MiB", "M", "m":
		return val * 1024 * 1024, nil
	case "GiB", "G", "g":
		return val * 1024 * 1024 * 1024, nil
	case "TiB", "T", "t":
		return val * 1024 * 1024 * 1024 * 1024, nil
	case "KB":
		return val * 1000, nil
	case "MB":
		return val * 1000 * 1000, nil
	case "GB":
		return val * 1000 * 1000 * 1000, nil
	case "TB":
		return val * 1000 * 1000 * 1000 * 1000, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}
}

// formatSize formats bytes into a human readable string
func formatSize(size float64) string {
	const (
		KiB = 1024
		MiB = 1024 * 1024
		GiB = 1024 * 1024 * 1024
		TiB = 1024 * 1024 * 1024 * 1024
	)

	switch {
	case size >= TiB:
		return fmt.Sprintf("%.2fTiB", size/TiB)
	case size >= GiB:
		return fmt.Sprintf("%.2fGiB", size/GiB)
	case size >= MiB:
		return fmt.Sprintf("%.2fMiB", size/MiB)
	case size >= KiB:
		return fmt.Sprintf("%.2fKiB", size/KiB)
	default:
		return fmt.Sprintf("%.2fB", size)
	}
}
