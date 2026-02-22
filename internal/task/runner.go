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

	ytDlpPath, err := m.deps.GetYtDlpPath()
	if err != nil {
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
	m.saveTask(task)

	// 如果是子任务且未指定格式ID，则获取视频信息以选择最佳分辨率
	if task.FormatID == "" {
		m.emitTaskLog(task.ID, "正在获取视频信息以选择最佳分辨率...", false)
		info, err := m.deps.GetVideoInfo(task.URL)
		if err != nil {
			m.mu.Lock()
			task.Status = models.TaskStatusError
			m.mu.Unlock()
			m.emitTaskLog(task.ID, "Error resolving video info: "+err.Error(), false)
			m.emitTaskUpdate(task)
			m.saveTask(task)
			return
		}

		if len(info.Qualities) > 0 {
			// Qualities are sorted by height desc, so first is best
			best := info.Qualities[0]

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
			m.saveTask(task)
			m.emitTaskUpdate(task)
		} else {
			m.emitTaskLog(task.ID, "无法获取分辨率列表，将尝试使用默认最佳格式", false)
		}
	}

	// Create output directory based on title
	sanitizedTitle := utils.SanitizeFileName(task.Title)
	outputDir := filepath.Join(task.Dir, sanitizedTitle)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		m.emitTaskLog(task.ID, "Failed to create directory: "+err.Error(), false)
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

	ffmpegPath, err := m.deps.GetFFmpegPath()
	ffmpegLocation := ""
	if err != nil {
		ffmpegLocation = filepath.Dir(ffmpegPath)
	}

	args := []string{
		"--newline",
		"--encoding", "utf-8",
		"--ffmpeg-location", ffmpegLocation,
		"-o", "%(title)s.%(ext)s",
		"-P", outputDir,
		"-f", format,
	}

	if limit := config.GetDownloadRateLimit(); limit != "" {
		args = append(args, "-r", limit)
	}

	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}

	if ua := config.GetUserAgent(); ua != "" {
		args = append(args, "--user-agent", ua)
	}
	if ref := config.GetReferer(); ref != "" {
		args = append(args, "--referer", ref)
	}
	if config.GetGeoBypass() {
		args = append(args, "--geo-bypass")
	} else {
		args = append(args, "--no-geo-bypass")
	}

	if cookieArgs := config.GetCookieArgs(); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}

	if task.Format != "original" {
		args = append(args, "--merge-output-format", task.Format)
	}

	args = append(args, task.URL)

	cmd := utils.CreateCommandContext(ctx, ytDlpPath, args...)

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
		m.saveTask(task)
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
	deletingRegex := regexp.MustCompile(`Deleting original file (.+?)(?: \(pass -k to keep\))?$`)

	var wg sync.WaitGroup
	wg.Add(2)

	// Progress tracking
	var completedBytes int64
	var currentPartBytes int64
	var currentPartDownloaded int64
	var currentPartLogged bool

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
				path := utils.NormalizePath(outputDir, filename)
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

				task.FilePath = matches[1]
			}

			if matches := mergerRegex.FindStringSubmatch(line); len(matches) > 1 {
				mergedPath := matches[1]
				task.FilePath = mergedPath
			}

			if matches := deletingRegex.FindStringSubmatch(line); len(matches) > 1 {
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
				m.emitTaskUpdate(task)
			}
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	if waitErr != nil {
		// Check for exit code 101 (Max downloads reached)
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) && exitErr.ExitCode() == 101 {
			task.Status = models.TaskStatusCompleted
			task.Progress = 100
			m.emitTaskLog(task.ID, "Download limit reached (expected)", false)
			if m.OnTaskComplete != nil {
				go m.OnTaskComplete(task)
			}
		} else {
			// Check if paused or deleted
			m.mu.Lock()
			isPaused := task.Status == models.TaskStatusPaused
			_, isDeleted := m.deletedTasks[task.ID]
			m.mu.Unlock()

			if isPaused || isDeleted {
				return
			}

			task.Status = models.TaskStatusError
			m.emitTaskLog(task.ID, "Exit Error: "+waitErr.Error(), false)
			if m.OnTaskFailed != nil {
				go m.OnTaskFailed(task)
			}
		}
	} else {
		task.Progress = 100
		task.FileExists = false

		if !task.FileExists && task.FilePath != "" {
			if _, statErr := os.Stat(task.FilePath); statErr == nil {
				task.FileExists = true
			}
		}

		if task.TrimMode != models.TrimModeNone && task.FileExists && task.FilePath != "" {
			task.Status = models.TaskStatusTrimming
			m.emitTaskUpdate(task)

			modeStr := "保留原文件"
			if task.TrimMode == models.TrimModeOverwrite {
				modeStr = "覆盖原文件"
			}
			m.emitTaskLog(task.ID, fmt.Sprintf("正在进行裁剪 (开始: %s, 结束: %s, 模式: %s)...", task.TrimStart, task.TrimEnd, modeStr), false)

			ext := filepath.Ext(task.FilePath)
			base := strings.TrimSuffix(task.FilePath, ext)
			trimmedPath := base + "_trimmed" + ext

			ffmpegPath, err := m.deps.GetFFmpegPath()
			if err != nil {
				m.emitTaskLog(task.ID, "FFmpeg not found", false)
				task.Status = models.TaskStatusTrimFailed
				m.emitTaskUpdate(task)
				m.saveTask(task)
				return
			}

			// args: -i input -ss start -to end -c copy output
			// Use -y to overwrite output if exists
			// Using output seeking (-ss after -i) to preserve time semantics (end time is absolute)
			args := []string{"-i", task.FilePath}
			if task.TrimStart != "" {
				args = append(args, "-ss", task.TrimStart)
			}
			if task.TrimEnd != "" {
				args = append(args, "-to", task.TrimEnd)
			}
			args = append(args, "-c", "copy", "-y", trimmedPath)

			cmd := utils.CreateCommandContext(ctx, ffmpegPath, args...)

			if output, err := cmd.CombinedOutput(); err != nil {
				m.emitTaskLog(task.ID, fmt.Sprintf("裁剪失败: %v, %s", err, string(output)), false)
				task.Status = models.TaskStatusTrimFailed
			} else {
				m.emitTaskLog(task.ID, "裁剪完成", false)

				if task.TrimMode == models.TrimModeOverwrite {
					if err := os.Remove(task.FilePath); err != nil {
						m.emitTaskLog(task.ID, "删除原文件失败: "+err.Error(), false)
						task.Status = models.TaskStatusTrimFailed
					} else {
						if err := os.Rename(trimmedPath, task.FilePath); err != nil {
							m.emitTaskLog(task.ID, "重命名裁剪文件失败: "+err.Error(), false)
							task.Status = models.TaskStatusTrimFailed
						} else {
							// Update file size in task
							if info, err := os.Stat(task.FilePath); err == nil {
								task.TotalSize = utils.FormatBytes(info.Size())
								task.TotalBytes = info.Size()
							}
							task.Status = models.TaskStatusCompleted
						}
					}
				} else {
					// Keep both
					task.Status = models.TaskStatusCompleted
				}
			}
		} else {
			task.Status = models.TaskStatusCompleted
			if m.OnTaskComplete != nil {
				go m.OnTaskComplete(task)
			}
		}
	}

	m.emitTaskUpdate(task)
	m.saveTask(task)
}
