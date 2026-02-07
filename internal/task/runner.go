package task

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"yt-downloader/internal/models"
	"yt-downloader/internal/utils"
)

func (m *Manager) processTask(ctx context.Context, task *models.DownloadTask) {
	defer func() {
		m.mu.Lock()
		delete(m.cancelFuncs, task.ID)
		m.mu.Unlock()
	}()

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
		"--ffmpeg-location", filepath.Dir(m.downloader.BinPath), // Explicitly set ffmpeg location
		"-o", "%(title)s.%(ext)s",
		"-P", task.Dir,
		"-f", format,
		"--playlist-items", "1",
	}

	if task.Format != "" {
		args = append(args, "--merge-output-format", task.Format)
	}

	args = append(args, task.URL)

	cmd := exec.CommandContext(ctx, m.downloader.BinPath, args...)

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

	var wg sync.WaitGroup
	wg.Add(2)

	// Progress tracking
	var completedBytes int64
	var currentPartBytes int64
	var currentPartDownloaded int64

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
				path := filename
				if !filepath.IsAbs(path) {
					path = filepath.Join(task.Dir, path)
				}
				if info, err := os.Stat(path); err == nil {
					completedBytes += info.Size()
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
			if destinationRegex.MatchString(line) {
				// If we were downloading a part, assume it finished?
				// No, destination is printed at START of download.
				// So if we had a previous part, add its size to completed.
				if currentPartBytes > 0 {
					completedBytes += currentPartBytes
				}
				currentPartBytes = 0
				currentPartDownloaded = 0
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
		task.FileExists = true
	}
	m.emitTaskUpdate(task)
	m.saveTasks()
}
