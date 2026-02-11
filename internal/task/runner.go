package task

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"Kairo/internal/config"
	"Kairo/internal/models"
	"Kairo/internal/utils"
)

func (m *Manager) processTask(ctx context.Context, task *models.DownloadTask) {
	defer func() {
		m.mu.Lock()
		delete(m.cancelFuncs, task.ID)
		if _, deleted := m.deletedTasks[task.ID]; deleted {
			delete(m.deletedTasks, task.ID)
		}
		m.mu.Unlock()
		go m.scheduleTasks()
	}()

	updateFileEntry := func(path string, sizeBytes int64, progress float64) {
		path = utils.NormalizePath(task.Dir, path)
		if path == "" {
			return
		}
		for i := range task.Files {
			if task.Files[i].Path == path {
				if sizeBytes > 0 {
					task.Files[i].SizeBytes = sizeBytes
					task.Files[i].Size = utils.FormatBytes(sizeBytes)
				}
				if progress >= 0 {
					task.Files[i].Progress = progress
				}
				return
			}
		}
		entry := models.DownloadFile{
			Path: path,
		}
		if sizeBytes > 0 {
			entry.SizeBytes = sizeBytes
			entry.Size = utils.FormatBytes(sizeBytes)
		}
		if progress >= 0 {
			entry.Progress = progress
		}
		task.Files = append(task.Files, entry)
	}

	removeFileEntry := func(path string) {
		path = utils.NormalizePath(task.Dir, path)
		if path == "" || len(task.Files) == 0 {
			return
		}
		next := task.Files[:0]
		for _, file := range task.Files {
			if file.Path != path {
				next = append(next, file)
			}
		}
		task.Files = next
	}

	m.downloader.EnsureYtDlp(m.assetProvider)
	if m.downloader.BinPath == "" {
		task.Status = models.TaskStatusError
		m.emitTaskLog(task.ID, "Error: yt-dlp not found", false)
		m.emitTaskUpdate(task)
		return
	}

	m.mu.Lock()
	task.Status = models.TaskStatusStarting
	task.Speed = "启动中..."
	task.Eta = ""
	m.mu.Unlock()

	m.emitTaskLog(task.ID, "正在启动下载引擎...", false)
	m.emitTaskUpdate(task)
	m.saveTasks()

	// 如果是子任务且未指定格式ID，则获取视频信息以选择最佳分辨率
	if task.ParentID != "" && task.FormatID == "" {
		m.emitTaskLog(task.ID, "正在获取视频信息以选择最佳分辨率...", false)
		info, err := m.downloader.GetVideoInfo(task.URL, m.assetProvider)
		if err != nil {
			m.mu.Lock()
			task.Status = models.TaskStatusError
			m.mu.Unlock()
			m.emitTaskLog(task.ID, "Error resolving video info: "+err.Error(), false)
			m.emitTaskUpdate(task)
			m.saveTasks()
			return
		}

		if len(info.Qualities) > 0 {
			// Qualities are sorted by height desc, so first is best
			best := info.Qualities[0]
			// If first is "Audio Only" and there are others, we might want video?
			// But "best" usually implies best video+audio.
			// Our downloader logic puts "Audio Only" at the end usually?
			// Let's check downloader.go sort logic.
			// It sorts heights desc. "Audio Only" is appended at the end.
			// So index 0 should be best video.
			// However, we should double check if it's audio only task?
			// task.Quality == "best" implies we want best available.

			task.FormatID = best.FormatID
			task.TotalBytes = best.TotalBytes
			task.TotalSize = utils.FormatBytes(best.TotalBytes)
			if task.Title == "" || task.Title == task.URL {
				task.Title = info.Title
			}
			if task.Thumbnail == "" {
				task.Thumbnail = info.Thumbnail
			}
			m.emitTaskLog(task.ID, fmt.Sprintf("已选择最佳分辨率: %s (%s)", best.Label, best.TotalSize), false)
			m.saveTasks()
			m.emitTaskUpdate(task)
		} else {
			m.emitTaskLog(task.ID, "无法获取分辨率列表，将尝试使用默认最佳格式", false)
		}
	}

	// Build args based on quality
	format := ""
	if task.FormatID != "" {
		format = task.FormatID
	} else {
		format = "bestvideo+bestaudio/best"
		switch task.Quality {
		case "4k":
			format = "bestvideo[height<=2160]+bestaudio/best[height<=2160]"
		case "2k":
			format = "bestvideo[height<=1440]+bestaudio/best[height<=1440]"
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
	}

	args := []string{
		"--newline",           // Important for parsing
		"--encoding", "utf-8", // Force UTF-8 output for correct parsing
		// "--js-runtimes", "node,deno", // Removed: auto-detection is preferred, or use specific path if needed
		"--ffmpeg-location", filepath.Dir(m.downloader.BinPath), // Explicitly set ffmpeg location
		"-o", "%(title)s.%(ext)s",
		"-P", task.Dir,
		"-f", format,
	}

	if limit := config.GetDownloadRateLimit(); limit != "" {
		args = append(args, "-r", limit)
	}

	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	if cookieArgs := config.GetCookieArgs(task.URL); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}

	if task.Format != "original" {
		args = append(args, "--merge-output-format", task.Format)
	}

	args = append(args, task.URL)

	cmd := exec.CommandContext(ctx, m.downloader.BinPath, args...)
	utils.HideWindow(cmd)

	// Separate pipes for stdout and stderr
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		m.mu.Lock()
		isPaused := task.Status == models.TaskStatusPaused
		m.mu.Unlock()
		if isPaused {
			return
		}

		task.Status = models.TaskStatusError
		m.emitTaskLog(task.ID, "Start Error: "+err.Error(), false)
		m.emitTaskUpdate(task)
		m.saveTasks()
		return
	}

	m.mu.Lock()
	task.Status = models.TaskStatusDownloading
	m.mu.Unlock()
	m.emitTaskLog(task.ID, "下载引擎已启动，正在解析...", false)

	// Regex for progress
	// [download]  25.0% of 10.00MiB at  1.00MiB/s ETA 00:05
	progressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+([~\d\.\w]+)(?:\s+at\s+([~\d\.\w/]+)\s+ETA\s+([\d:]+))?`)
	destinationRegex := regexp.MustCompile(`\[download\] Destination: (.+)`)
	alreadyDownloadedRegex := regexp.MustCompile(`\[download\] (.+) has already been downloaded`)
	mergerRegex := regexp.MustCompile(`\[Merger\] Merging formats into "(.+)"`)
	deletingRegex := regexp.MustCompile(`Deleting original file (.+)`)

	var wg sync.WaitGroup
	wg.Add(2)

	// Progress tracking
	var completedBytes int64
	var currentPartBytes int64
	var currentPartDownloaded int64
	var currentPartLogged bool
	var currentPartPath string

	// Stdout reader
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			isProgress := strings.HasPrefix(line, "[download]") && strings.Contains(line, "%")
			m.emitTaskLog(task.ID, line, isProgress)

			// Parse already downloaded
			if matches := alreadyDownloadedRegex.FindStringSubmatch(line); len(matches) > 1 {
				// File exists, try to get size
				filename := matches[1]
				// If path is relative, join with task.Dir?
				// yt-dlp usually outputs full path or relative to cwd.
				// We set -P task.Dir.
				path := utils.NormalizePath(task.Dir, filename)
				if info, err := os.Stat(path); err == nil {
					completedBytes += info.Size()
					updateFileEntry(path, info.Size(), 100)
				}

				// Re-calculate progress?
				if task.TotalBytes > 0 {
					p := float64(completedBytes+currentPartDownloaded) / float64(task.TotalBytes) * 100
					if p > 100 {
						p = 100
					}
					task.Progress = p
				}
				m.emitTaskUpdate(task)
			}

			// Parse Destination (new part starting)
			if matches := destinationRegex.FindStringSubmatch(line); len(matches) > 1 {
				// If we were downloading a part, assume it finished?
				// No, destination is printed at START of download.
				// So if we had a previous part, add its size to completed.
				if currentPartBytes > 0 {
					completedBytes += currentPartBytes
				}
				currentPartBytes = 0
				currentPartDownloaded = 0
				currentPartLogged = false

				currentPartPath = matches[1]
				task.FilePath = currentPartPath
				updateFileEntry(currentPartPath, 0, 0)
			}

			if matches := mergerRegex.FindStringSubmatch(line); len(matches) > 1 {
				mergedPath := matches[1]
				task.FilePath = mergedPath
				updateFileEntry(mergedPath, 0, 0)
			}

			if matches := deletingRegex.FindStringSubmatch(line); len(matches) > 1 {
				deletedPath := matches[1]
				removeFileEntry(deletedPath)
				m.emitTaskUpdate(task)
			}

			if strings.HasPrefix(line, "[Merger]") && task.Status != models.TaskStatusMerging {
				task.Status = models.TaskStatusMerging
				m.emitTaskUpdate(task)
			}

			// Parse progress
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 2 {
				percent, _ := strconv.ParseFloat(matches[1], 64)
				sizeStr := matches[2]
				speed := matches[3]
				eta := matches[4]

				task.Speed = speed
				task.Eta = eta

				// Parse size
				size, _ := utils.ParseSize(sizeStr)
				if size > 0 {
					currentPartBytes = int64(size)
					currentPartDownloaded = int64(percent / 100 * size)
				}
				if currentPartPath != "" {
					updateFileEntry(currentPartPath, currentPartBytes, percent)
				}

				// Calculate total progress
				if task.TotalBytes > 0 {
					totalDownloaded := completedBytes + currentPartDownloaded
					p := float64(totalDownloaded) / float64(task.TotalBytes) * 100
					if p > 100 {
						p = 100
					}
					task.Progress = p
				} else {
					// Fallback if total bytes unknown: just show current part percent?
					// Or 0?
					// Let's show current part percent / number of parts? No.
					// Just use the parsed percent as fallback?
					task.Progress = percent
				}

				if percent >= 100 && !currentPartLogged {
					m.emitTaskLog(task.ID, line, false)
					currentPartLogged = true
				}

				m.emitTaskUpdate(task)
			}
		}
	}()

	// Stderr reader
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			m.emitTaskLog(task.ID, line, false)
			if matches := deletingRegex.FindStringSubmatch(line); len(matches) > 1 {
				deletedPath := matches[1]
				removeFileEntry(deletedPath)
				m.emitTaskUpdate(task)
			}
		}
	}()

	err := cmd.Wait()
	wg.Wait()

	if err != nil {
		// Check for exit code 101 (Max downloads reached)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 101 {
			task.Status = models.TaskStatusCompleted
			task.Progress = 100
			m.emitTaskLog(task.ID, "Download limit reached (expected)", false)
		} else {
			// Check if paused
			m.mu.Lock()
			isPaused := task.Status == models.TaskStatusPaused
			m.mu.Unlock()

			if isPaused {
				return
			}

			task.Status = models.TaskStatusError
			m.emitTaskLog(task.ID, "Exit Error: "+err.Error(), false)
		}
	} else {
		task.Status = models.TaskStatusCompleted
		task.Progress = 100
		task.FileExists = false
	}
	if task.Status == models.TaskStatusCompleted {
		if task.FilePath != "" {
			updateFileEntry(task.FilePath, 0, 100)
		}
		for i := range task.Files {
			if info, statErr := os.Stat(task.Files[i].Path); statErr == nil {
				task.Files[i].SizeBytes = info.Size()
				task.Files[i].Size = utils.FormatBytes(info.Size())
				task.Files[i].Progress = 100
				task.FileExists = true
			}
		}
		if !task.FileExists && task.FilePath != "" {
			if _, statErr := os.Stat(task.FilePath); statErr == nil {
				task.FileExists = true
			}
		}
	}
	m.emitTaskUpdate(task)
	m.saveTasks()
}
