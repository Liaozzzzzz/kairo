package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/config"
	"Kairo/internal/deps"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Manager struct {
	ctx       context.Context
	db        *sql.DB
	aiService *ai.Manager
	deps      *deps.Manager
}

func NewManager(ctx context.Context, db *sql.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:       ctx,
		db:        db,
		aiService: ai.NewManager(ctx),
		deps:      d,
	}
	m.initTable()
	return m
}

func (m *Manager) initTable() {
	if m.db == nil {
		return
	}

	createVideosTableSQL := `CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		task_id TEXT,
		title TEXT,
		url TEXT,
		file_path TEXT,
		thumbnail TEXT,
		duration REAL,
		size INTEGER,
		format TEXT,
		resolution TEXT,
		created_at INTEGER,
		description TEXT,
		uploader TEXT,
		subtitles TEXT,
		summary TEXT,
		tags TEXT,
		evaluation TEXT,
		status TEXT
	);`

	_, err := m.db.Exec(createVideosTableSQL)
	if err != nil {
		fmt.Printf("Failed to create videos table: %v\n", err)
	}

	createHighlightsTableSQL := `CREATE TABLE IF NOT EXISTS video_highlights (
		id TEXT PRIMARY KEY,
		video_id TEXT,
		start_time TEXT,
		end_time TEXT,
		description TEXT,
		file_path TEXT
	);`

	_, err = m.db.Exec(createHighlightsTableSQL)
	if err != nil {
		fmt.Printf("Failed to create highlights table: %v\n", err)
	}
}

func (m *Manager) SaveVideo(v *models.Video) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `INSERT OR REPLACE INTO videos (
		id, task_id, title, url, file_path, thumbnail, duration, size, format, resolution, created_at,
		description, uploader, subtitles, summary, tags, evaluation, status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	subtitlesJSON, _ := json.Marshal(v.Subtitles)
	tagsJSON, _ := json.Marshal(v.Tags)

	_, err := m.db.Exec(query,
		v.ID, v.TaskID, v.Title, v.URL, v.FilePath, v.Thumbnail, v.Duration, v.Size, v.Format, v.Resolution, v.CreatedAt,
		v.Description, v.Uploader, string(subtitlesJSON), v.Summary, string(tagsJSON), v.Evaluation, v.Status,
	)

	return err
}

func (m *Manager) CreateFromTask(t *models.DownloadTask) error {
	if t.FilePath == "" {
		return fmt.Errorf("task has no file path")
	}
	exists, err := m.HasVideoForTask(t.ID, t.FilePath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	v := &models.Video{
		ID:         uuid.New().String(),
		TaskID:     t.ID,
		Title:      t.Title,
		URL:        t.URL,
		FilePath:   t.FilePath,
		Thumbnail:  t.Thumbnail,
		Size:       t.TotalBytes,
		Format:     t.Format,
		Resolution: t.Quality,
		CreatedAt:  time.Now().Unix(),
		Status:     "none",
	}

	// Scan for subtitles
	dir := filepath.Dir(t.FilePath)
	ext := filepath.Ext(t.FilePath)
	base := strings.TrimSuffix(filepath.Base(t.FilePath), ext)

	matches := scanSubtitles(dir, base)

	for _, match := range matches {
		v.Subtitles = append(v.Subtitles, match)
	}

	if v.Duration <= 0 && v.FilePath != "" {
		if duration, err := m.getDurationFromFile(v.FilePath); err == nil {
			v.Duration = duration
		}
	}

	err = m.SaveVideo(v)
	if err == nil && len(v.Subtitles) == 0 && v.URL != "" {
		go m.FetchSubtitles(v.ID)
	}
	return err
}

func scanSubtitles(dir, base string) []string {
	matches, _ := filepath.Glob(filepath.Join(dir, base+".*.vtt"))
	srtMatches, _ := filepath.Glob(filepath.Join(dir, base+".*.srt"))
	matches = append(matches, srtMatches...)
	return matches
}

func (m *Manager) getDurationFromFile(filePath string) (float64, error) {
	if filePath == "" {
		return 0, fmt.Errorf("file path is empty")
	}
	ffmpegPath, err := m.deps.GetFFmpegPath()
	if err != nil {
		return 0, err
	}
	cmd := utils.CreateCommand(ffmpegPath, "-i", filePath)
	output, cmdErr := cmd.CombinedOutput()
	re := regexp.MustCompile(`Duration:\s+(\d+):(\d+):(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) != 4 {
		if cmdErr != nil {
			return 0, cmdErr
		}
		return 0, fmt.Errorf("duration not found")
	}
	hours, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return 0, err
	}
	return float64(hours*3600+minutes*60) + seconds, nil
}

func (m *Manager) HasVideoForTask(taskID, filePath string) (bool, error) {
	if m.db == nil {
		return false, fmt.Errorf("database not initialized")
	}
	if taskID == "" && filePath == "" {
		return false, nil
	}

	query := "SELECT 1 FROM videos WHERE "
	var args []interface{}
	if taskID != "" && filePath != "" {
		query += "task_id = ? OR file_path = ? LIMIT 1"
		args = append(args, taskID, filePath)
	} else if taskID != "" {
		query += "task_id = ? LIMIT 1"
		args = append(args, taskID)
	} else {
		query += "file_path = ? LIMIT 1"
		args = append(args, filePath)
	}

	var exists int
	err := m.db.QueryRow(query, args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) GetVideos(filter models.VideoFilter) ([]*models.Video, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := "SELECT * FROM videos WHERE 1=1"
	var args []interface{}

	if filter.Status != "" && filter.Status != "all" {
		if filter.Status == "analyzed" {
			query += " AND status = 'completed'"
		} else if filter.Status == "unanalyzed" {
			query += " AND (status IS NULL OR status = '' OR status = 'none')"
		}
	}

	if filter.Query != "" {
		query += " AND title LIKE ?"
		args = append(args, "%"+filter.Query+"%")
	}

	query += " ORDER BY created_at DESC"
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*models.Video
	for rows.Next() {
		var v models.Video
		var subtitlesJSON, tagsJSON string
		var url sql.NullString
		err := rows.Scan(
			&v.ID, &v.TaskID, &v.Title, &url, &v.FilePath, &v.Thumbnail, &v.Duration, &v.Size, &v.Format, &v.Resolution, &v.CreatedAt,
			&v.Description, &v.Uploader, &subtitlesJSON, &v.Summary, &tagsJSON, &v.Evaluation, &v.Status,
		)
		if err != nil {
			continue
		}
		if url.Valid {
			v.URL = url.String
		}
		json.Unmarshal([]byte(subtitlesJSON), &v.Subtitles)
		json.Unmarshal([]byte(tagsJSON), &v.Tags)

		videos = append(videos, &v)
	}

	return videos, nil
}

func (m *Manager) GetVideo(id string) (*models.Video, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var v models.Video
	var subtitlesJSON, tagsJSON string
	var url sql.NullString
	// Explicitly select columns to avoid issues with old schema having extra columns
	query := `SELECT id, task_id, title, url, file_path, thumbnail, duration, size, format, resolution, created_at,
		description, uploader, subtitles, summary, tags, evaluation, status
		FROM videos WHERE id = ?`

	err := m.db.QueryRow(query, id).Scan(
		&v.ID, &v.TaskID, &v.Title, &url, &v.FilePath, &v.Thumbnail, &v.Duration, &v.Size, &v.Format, &v.Resolution, &v.CreatedAt,
		&v.Description, &v.Uploader, &subtitlesJSON, &v.Summary, &tagsJSON, &v.Evaluation, &v.Status,
	)
	if err != nil {
		return nil, err
	}
	if url.Valid {
		v.URL = url.String
	}
	json.Unmarshal([]byte(subtitlesJSON), &v.Subtitles)
	json.Unmarshal([]byte(tagsJSON), &v.Tags)

	return &v, nil
}

func (m *Manager) GetHighlights(videoID string) ([]models.AIHighlight, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := "SELECT id, video_id, start_time, end_time, description, file_path FROM video_highlights WHERE video_id = ?"
	rows, err := m.db.Query(query, videoID)
	if err != nil {
		return []models.AIHighlight{}, nil // Return empty if table doesn't exist or error
	}
	defer rows.Close()

	var highlights []models.AIHighlight
	for rows.Next() {
		var h models.AIHighlight
		var filePath sql.NullString
		if err := rows.Scan(&h.ID, &h.VideoID, &h.Start, &h.End, &h.Description, &filePath); err != nil {
			continue
		}
		if filePath.Valid {
			h.FilePath = filePath.String
		}
		highlights = append(highlights, h)
	}
	return highlights, nil
}

// UpdateHighlightFilePath updates the file path for a specific highlight
func (m *Manager) UpdateHighlightFilePath(highlightID, filePath string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := m.db.Exec("UPDATE video_highlights SET file_path = ? WHERE id = ?", filePath, highlightID)
	return err
}

func (m *Manager) DeleteVideo(id string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec("DELETE FROM video_highlights WHERE video_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete highlights: %v", err)
	}

	_, err = tx.Exec("DELETE FROM videos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete video: %v", err)
	}

	return tx.Commit()
}

func (m *Manager) FetchSubtitles(id string) error {
	v, err := m.GetVideo(id)
	if err != nil {
		return err
	}
	if v.URL == "" {
		return fmt.Errorf("no URL found for video %s", id)
	}
	ytDlpPath, err := m.deps.GetYtDlpPath()
	if err != nil {
		return err
	}

	// Prepare command
	dir := filepath.Dir(v.FilePath)
	ext := filepath.Ext(v.FilePath)
	baseName := strings.TrimSuffix(filepath.Base(v.FilePath), ext)
	outputTemplate := filepath.Join(dir, baseName+".%(ext)s")

	args := []string{
		"--skip-download",
		"--write-subs",
		"--write-auto-subs",
		"-o", outputTemplate,
	}

	if proxy := config.GetProxyUrl(); proxy != "" {
		args = append(args, "--proxy", proxy)
	}
	if ua := config.GetUserAgent(); ua != "" {
		args = append(args, "--user-agent", ua)
	}
	if cookieArgs := config.GetCookieArgs(); len(cookieArgs) > 0 {
		args = append(args, cookieArgs...)
	}

	args = append(args, v.URL)
	wailsRuntime.LogInfo(m.ctx, fmt.Sprintf("running command: %s %s", ytDlpPath, strings.Join(args, " ")))
	cmd := utils.CreateCommandContext(m.ctx, ytDlpPath, args...)

	if output, err := cmd.CombinedOutput(); err != nil {
		wailsRuntime.LogError(m.ctx, fmt.Sprintf("failed to fetch subtitles: %v, output: %s", err, string(output)))
		return fmt.Errorf("failed to fetch subtitles: %v, output: %s", err, string(output))
	}

	matches := scanSubtitles(dir, baseName)
	if len(matches) == 0 {
		asrMatches, err := m.GenerateSubtitlesByASR(v, dir)
		if err != nil {
			wailsRuntime.LogError(m.ctx, fmt.Sprintf("failed to generate subtitles by ASR: %v", err))
		}
		if len(asrMatches) > 0 {
			matches = asrMatches
		}
	}

	v.Subtitles = matches

	// Update DB
	if err := m.SaveVideo(v); err != nil {
		return err
	}

	// Emit update event
	wailsRuntime.EventsEmit(m.ctx, "video:updated", v)
	return nil
}

func (m *Manager) GenerateSubtitlesByASR(v *models.Video, dir string) ([]string, error) {
	wailsRuntime.LogInfo(m.ctx, "generating subtitles by ASR API")
	inputPath := v.FilePath
	tempPath := ""
	if strings.ToLower(filepath.Ext(v.FilePath)) != ".mp3" {
		ffmpegPath, err := m.deps.GetFFmpegPath()
		if err != nil {
			return nil, fmt.Errorf("asr failed: %v", err)
		}
		tempDir := filepath.Dir(v.FilePath)
		tempFile, err := os.CreateTemp(tempDir, "whisper-*.mp3")
		if err != nil {
			return nil, err
		}
		tempPath = tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return nil, err
		}
		args := []string{"-i", v.FilePath, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "libmp3lame", "-y", tempPath}
		cmd := utils.CreateCommand(ffmpegPath, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			_ = os.Remove(tempPath)
			return nil, fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
		}
		inputPath = tempPath
	}
	if tempPath != "" {
		defer os.Remove(tempPath)
	}
	content, err := m.aiService.TranscribeWhisper(inputPath, "vtt")
	if err != nil {
		if errors.Is(err, ai.ErrWhisperDisabled) {
			wailsRuntime.LogInfo(m.ctx, "Whisper disabled, skip generating subtitles")
			return nil, nil
		}
		return nil, fmt.Errorf("asr failed: %v", err)
	}

	ext := filepath.Ext(v.FilePath)
	baseName := strings.TrimSuffix(filepath.Base(v.FilePath), ext)
	outputPath := filepath.Join(dir, baseName+".asr.vtt")
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return scanSubtitles(dir, baseName), nil
}

func (m *Manager) UpdateVideoStatus(id, status, summary, evaluation string, tags []string, highlights []models.AIHighlight) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	tagsJSON, _ := json.Marshal(tags)
	// highlights are now saved separately, but we might want to update them if status is completed

	query := "UPDATE videos SET status = ?, summary = ?, evaluation = ?, tags = ? WHERE id = ?"
	_, err := m.db.Exec(query, status, summary, evaluation, string(tagsJSON), id)

	if err == nil {
		// If completed, save highlights
		if status == "completed" && len(highlights) > 0 {
			// Delete old highlights
			_, _ = m.db.Exec("DELETE FROM video_highlights WHERE video_id = ?", id)

			highlightQuery := `INSERT INTO video_highlights (id, video_id, start_time, end_time, description, file_path) VALUES (?, ?, ?, ?, ?, ?)`
			for _, h := range highlights {
				hID := h.ID
				if hID == "" {
					hID = uuid.New().String()
				}
				_, _ = m.db.Exec(highlightQuery, hID, id, h.Start, h.End, h.Description, h.FilePath)
			}
		}

		// Emit event for frontend
		wailsRuntime.EventsEmit(m.ctx, "video:ai_status", map[string]interface{}{
			"id":         id,
			"status":     status,
			"summary":    summary,
			"evaluation": evaluation,
			"tags":       tags,
			"highlights": highlights,
		})
	}

	return err
}

func (m *Manager) AnalyzeVideo(id string) error {
	v, err := m.GetVideo(id)
	if err != nil {
		return err
	}

	// Set status to analyzing
	// If re-analyzing, we clear previous results but keep metadata
	m.UpdateVideoStatus(id, "processing", "", "", nil, nil)

	go func() {
		var subtitlesContent string
		if len(v.Subtitles) > 0 {
			if segments, err := parseSubtitleFile(v.Subtitles[0]); err == nil && len(segments) > 0 {
				subtitlesContent = buildSubtitleText(segments)
			} else if content, readErr := os.ReadFile(v.Subtitles[0]); readErr == nil {
				subtitlesContent = string(content)
			}
		}

		meta := ai.VideoMetadata{
			Title:       v.Title,
			Description: v.Description,
			Subtitles:   subtitlesContent,
			Uploader:    v.Uploader,
			Duration:    utils.FormatDuration(v.Duration),
			Resolution:  v.Resolution,
			Format:      v.Format,
			Size:        utils.FormatBytes(v.Size),
			Date:        time.Unix(v.CreatedAt, 0).Format("2006-01-02"),
		}

		result, err := m.aiService.Analyze(meta)

		if err != nil {
			if errors.Is(err, ai.ErrAIDisabled) {
				m.UpdateVideoStatus(id, "failed", "AI is disabled in settings", "", nil, nil)
				return
			}
			fmt.Printf("AI Analysis failed: %v\n", err)
			m.UpdateVideoStatus(id, "failed", fmt.Sprintf("Error: %v", err), "", nil, nil)
			return
		}

		// Convert result highlights to model highlights
		var highlights []models.AIHighlight
		for _, h := range result.Highlights {
			highlights = append(highlights, models.AIHighlight{
				ID:          uuid.New().String(),
				VideoID:     id,
				Start:       h.Start,
				End:         h.End,
				Description: h.Description,
			})
		}

		m.UpdateVideoStatus(id, "completed", result.Summary, result.Evaluation, result.Tags, highlights)

		// Async Clip Highlights
		if len(highlights) > 0 {
			go m.ClipHighlights(id, highlights)
		}
	}()

	return nil
}

func (m *Manager) ClipHighlights(videoID string, highlights []models.AIHighlight) {
	v, err := m.GetVideo(videoID)
	if err != nil {
		fmt.Printf("Failed to get video for clipping: %v\n", err)
		return
	}

	ffmpegPath, err := m.deps.GetFFmpegPath()
	if err != nil {
		fmt.Printf("Failed to get ffmpeg: %v\n", err)
		return
	}

	dir := filepath.Dir(v.FilePath)
	ext := filepath.Ext(v.FilePath)
	baseName := strings.TrimSuffix(filepath.Base(v.FilePath), ext)

	for _, h := range highlights {
		safeStart := strings.ReplaceAll(h.Start, ":", "-")
		safeEnd := strings.ReplaceAll(h.End, ":", "-")
		outputName := fmt.Sprintf("%s_clip_%s_%s%s", baseName, safeStart, safeEnd, ext)
		outputPath := filepath.Join(dir, outputName)

		args := []string{"-i", v.FilePath, "-ss", h.Start, "-to", h.End, "-c", "copy", "-y", outputPath}

		cmd := utils.CreateCommand(ffmpegPath, args...)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Failed to clip highlight %s: %v, output: %s\n", h.ID, err, string(output))
			continue
		}

		// Update DB with file path
		_ = m.UpdateHighlightFilePath(h.ID, outputPath)
	}

	// Notify frontend again with updated file paths
	// Re-fetch to get file paths
	updatedHighlights, _ := m.GetHighlights(videoID)

	wailsRuntime.EventsEmit(m.ctx, "video:ai_status", map[string]interface{}{
		"id":         v.ID,
		"status":     v.Status,
		"summary":    v.Summary,
		"evaluation": v.Evaluation,
		"tags":       v.Tags,
		"highlights": updatedHighlights,
	})
}
