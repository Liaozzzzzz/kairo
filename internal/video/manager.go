package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/deps"
	"Kairo/internal/models"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Manager struct {
	ctx           context.Context
	db            *sql.DB
	aiService     *ai.Manager
	deps          *deps.Manager
	subtitleQueue chan SubtitleTask
}

func NewManager(ctx context.Context, db *sql.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:       ctx,
		db:        db,
		aiService: ai.NewManager(ctx),
		deps:      d,
	}
	m.InitSubtitleQueue()
	return m
}

func (m *Manager) SaveVideo(v *models.Video) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `INSERT OR REPLACE INTO videos (
		id, task_id, title, url, file_path, thumbnail, duration, size, format, resolution, created_at,
		description, uploader, summary, tags, evaluation, category_id, status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	tagsJSON, _ := json.Marshal(v.Tags)

	_, err := m.db.Exec(query,
		v.ID, v.TaskID, v.Title, v.URL, v.FilePath, v.Thumbnail, v.Duration, v.Size, v.Format, v.Resolution, v.CreatedAt,
		v.Description, v.Uploader, v.Summary, string(tagsJSON), v.Evaluation, v.CategoryID, v.Status,
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
		CategoryID: t.CategoryID,
		Status:     "none",
	}

	if v.Duration <= 0 && v.FilePath != "" {
		log.Printf("[CreateFromTask] get duration from file: %s", v.FilePath)
		if duration, err := m.getDurationFromFile(v.FilePath); err == nil {
			v.Duration = duration
		}
		log.Printf("[CreateFromTask] duration: %f", v.Duration)
	}

	err = m.SaveVideo(v)
	if err == nil {
		go m.FetchSubtitles(v.ID)
	}
	return err
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
		var tagsJSON string
		var url sql.NullString
		err := rows.Scan(
			&v.ID, &v.TaskID, &v.Title, &url, &v.FilePath, &v.Thumbnail, &v.Duration, &v.Size, &v.Format, &v.Resolution, &v.CreatedAt,
			&v.Description, &v.Uploader, &v.Summary, &tagsJSON, &v.Evaluation, &v.CategoryID, &v.Status,
		)
		if err != nil {
			continue
		}
		if url.Valid {
			v.URL = url.String
		}
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
	var tagsJSON string
	var url sql.NullString
	// Explicitly select columns to avoid issues with old schema having extra columns
	query := `SELECT id, task_id, title, url, file_path, thumbnail, duration, size, format, resolution, created_at,
		description, uploader, summary, tags, evaluation, category_id, status
		FROM videos WHERE id = ?`

	err := m.db.QueryRow(query, id).Scan(
		&v.ID, &v.TaskID, &v.Title, &url, &v.FilePath, &v.Thumbnail, &v.Duration, &v.Size, &v.Format, &v.Resolution, &v.CreatedAt,
		&v.Description, &v.Uploader, &v.Summary, &tagsJSON, &v.Evaluation, &v.CategoryID, &v.Status,
	)
	if err != nil {
		return nil, err
	}
	if url.Valid {
		v.URL = url.String
	}
	json.Unmarshal([]byte(tagsJSON), &v.Tags)

	return &v, nil
}

func (m *Manager) getCategoryPrompt(categoryID string) (string, error) {
	if m.db == nil || strings.TrimSpace(categoryID) == "" {
		return "", nil
	}
	var prompt sql.NullString
	err := m.db.QueryRow("SELECT prompt FROM categories WHERE id = ?", categoryID).Scan(&prompt)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return prompt.String, nil
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

	subs, _ := m.GetVideoSubtitles(id)
	for _, sub := range subs {
		if strings.TrimSpace(sub.FilePath) == "" {
			continue
		}
		_ = utils.DeleteFile(sub.FilePath)
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

	_, err = tx.Exec("DELETE FROM video_subtitles WHERE video_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete subtitles: %v", err)
	}

	_, err = tx.Exec("DELETE FROM videos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete video: %v", err)
	}

	return tx.Commit()
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
			_ = m.clearHighlightFiles(id)

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
		var subtitleStats string
		var energyCandidatesText string
		var energyCandidates []energyCandidate
		var subtitleSegments []subtitleSegment
		subtitlePath := ""
		if subs, err := m.GetVideoSubtitles(v.ID); err == nil {
			for _, sub := range subs {
				if sub.Status == models.SubtitleStatusSuccess && strings.TrimSpace(sub.FilePath) != "" {
					subtitlePath = sub.FilePath
					break
				}
			}
		}

		if strings.TrimSpace(subtitlePath) != "" {
			if segments, err := parseSubtitleFile(subtitlePath); err == nil && len(segments) > 0 {
				subtitleSegments = segments
				subtitlesContent = buildSubtitleText(segments)
				subtitleStats, energyCandidates = buildSubtitleAnalysis(segments, v.Duration)
				energyCandidatesText = formatEnergyCandidates(energyCandidates)
			} else if content, readErr := os.ReadFile(subtitlePath); readErr == nil {
				subtitlesContent = string(content)
			}
		}
		if len(subtitlesContent) > 12000 {
			head := subtitlesContent[:8000]
			tail := subtitlesContent[len(subtitlesContent)-4000:]
			subtitlesContent = head + "\n...\n" + tail
		}

		meta := ai.VideoMetadata{
			Title:            v.Title,
			Description:      v.Description,
			Subtitles:        subtitlesContent,
			SubtitleStats:    subtitleStats,
			EnergyCandidates: energyCandidatesText,
			Uploader:         v.Uploader,
			Duration:         utils.FormatDuration(v.Duration),
			Resolution:       v.Resolution,
			Format:           v.Format,
			Size:             utils.FormatBytes(v.Size),
			Date:             time.Unix(v.CreatedAt, 0).Format("2006-01-02"),
		}

		categoryPrompt, _ := m.getCategoryPrompt(v.CategoryID)
		result, err := m.aiService.Analyze(meta, categoryPrompt)

		if err != nil {
			if errors.Is(err, ai.ErrAIDisabled) {
				m.UpdateVideoStatus(id, "failed", "AI is disabled in settings", "", nil, nil)
				return
			}
			fmt.Printf("AI Analysis failed: %v\n", err)
			m.UpdateVideoStatus(id, "failed", fmt.Sprintf("Error: %v", err), "", nil, nil)
			return
		}

		if len(result.Highlights) == 0 && len(energyCandidates) > 0 {
			result.Highlights = buildFallbackHighlights(energyCandidates)
		}
		result.Highlights = normalizeHighlights(result.Highlights, v.Duration, subtitleSegments, energyCandidates)

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

func (m *Manager) clearHighlightFiles(videoID string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	rows, err := m.db.Query("SELECT file_path FROM video_highlights WHERE video_id = ?", videoID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var filePath sql.NullString
		if err := rows.Scan(&filePath); err != nil {
			continue
		}
		if !filePath.Valid || strings.TrimSpace(filePath.String) == "" {
			continue
		}
		if err := utils.DeleteFile(filePath.String); err != nil {
			fmt.Printf("Failed to remove highlight file %s: %v\n", filePath.String, err)
		}
	}

	_, err = m.db.Exec("DELETE FROM video_highlights WHERE video_id = ?", videoID)
	return err
}

func buildFallbackHighlights(candidates []energyCandidate) []struct {
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
} {
	highlights := make([]struct {
		Start       string `json:"start"`
		End         string `json:"end"`
		Description string `json:"description"`
	}, 0, len(candidates))
	for _, candidate := range candidates {
		highlights = append(highlights, struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}{
			Start:       formatTimestampHMS(candidate.Start),
			End:         formatTimestampHMS(candidate.End),
			Description: "高能片段：" + candidate.Reason,
		})
	}
	return highlights
}

type highlightRange struct {
	Start       float64
	End         float64
	Description string
}

func normalizeHighlights(highlights []struct {
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
}, videoDuration float64, segments []subtitleSegment, candidates []energyCandidate) []struct {
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
} {
	if len(highlights) == 0 {
		return highlights
	}
	maxTime := videoDuration
	if maxTime <= 0 {
		maxTime = maxSegmentEnd(segments)
	}
	if maxTime <= 0 && len(candidates) > 0 {
		maxTime = candidates[len(candidates)-1].End
	}
	minDuration := 60.0
	targetDuration := 90.0
	maxDuration := 180.0
	var normalized []highlightRange
	for _, h := range highlights {
		startSec, okStart := parseHmsTimestamp(h.Start)
		endSec, okEnd := parseHmsTimestamp(h.End)
		if !okStart || !okEnd || endSec <= startSec {
			continue
		}
		start := startSec
		end := endSec
		duration := end - start
		if duration < minDuration {
			mid := (start + end) / 2
			start = mid - targetDuration/2
			end = mid + targetDuration/2
		} else if duration > maxDuration {
			mid := (start + end) / 2
			start = mid - maxDuration/2
			end = mid + maxDuration/2
		}
		start, end = clampRange(start, end, maxTime)
		start, end = snapRangeToSegments(segments, start, end)
		start, end = alignToEnergyCandidate(start, end, candidates, minDuration, maxTime)
		normalized = append(normalized, highlightRange{
			Start:       start,
			End:         end,
			Description: h.Description,
		})
	}
	if len(normalized) == 0 {
		return highlights
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Start < normalized[j].Start
	})
	merged := mergeHighlightRanges(normalized, 5.0)
	final := make([]struct {
		Start       string `json:"start"`
		End         string `json:"end"`
		Description string `json:"description"`
	}, 0, len(merged))
	for _, h := range merged {
		final = append(final, struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}{
			Start:       formatTimestampHMS(h.Start),
			End:         formatTimestampHMS(h.End),
			Description: h.Description,
		})
	}
	return final
}

func alignToEnergyCandidate(start float64, end float64, candidates []energyCandidate, minDuration float64, maxTime float64) (float64, float64) {
	if len(candidates) == 0 {
		return start, end
	}
	bestOverlap := 0.0
	bestCandidate := energyCandidate{}
	for _, c := range candidates {
		overlap := math.Min(end, c.End) - math.Max(start, c.Start)
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestCandidate = c
		}
	}
	if bestOverlap <= 0 {
		return start, end
	}
	newStart := math.Min(start, bestCandidate.Start)
	newEnd := math.Max(end, bestCandidate.End)
	if newEnd-newStart < minDuration {
		mid := (newStart + newEnd) / 2
		newStart = mid - minDuration/2
		newEnd = mid + minDuration/2
	}
	return clampRange(newStart, newEnd, maxTime)
}

func mergeHighlightRanges(items []highlightRange, gap float64) []highlightRange {
	if len(items) == 0 {
		return items
	}
	merged := []highlightRange{items[0]}
	for i := 1; i < len(items); i++ {
		last := &merged[len(merged)-1]
		if items[i].Start <= last.End+gap {
			if items[i].End > last.End {
				last.End = items[i].End
			}
			if items[i].Description != "" && items[i].Description != last.Description {
				last.Description = last.Description + "；" + items[i].Description
			}
			continue
		}
		merged = append(merged, items[i])
	}
	return merged
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

func (m *Manager) UpdateSubtitle(subtitleID string, content string, language string) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sub, err := m.getSubtitleByID(subtitleID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is empty")
	}

	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = sub.Language // fallback to existing language if not provided
	}

	if lang != sub.Language {
		// Language changed, generate new path
		video, err := m.GetVideo(sub.VideoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get video: %v", err)
		}

		// Use ensureUniqueSubtitlePath to generate new path
		newPath := ensureUniqueSubtitlePath(video.FilePath, lang, "manual")

		if err := os.WriteFile(newPath, []byte(content), 0o644); err != nil {
			return nil, err
		}

		// Delete old file
		if err := utils.DeleteFile(sub.FilePath); err != nil {
			log.Printf("Failed to delete old subtitle file %s: %v", sub.FilePath, err)
			// Non-fatal, continue
		}

		sub.FilePath = newPath
		sub.Language = lang
	} else {
		// Language same, just overwrite
		if err := os.WriteFile(sub.FilePath, []byte(content), 0o644); err != nil {
			return nil, err
		}
	}

	sub.UpdatedAt = time.Now().UnixMilli()
	_, err = m.db.Exec("UPDATE video_subtitles SET updated_at = ?, file_path = ?, language = ? WHERE id = ?",
		sub.UpdatedAt, sub.FilePath, sub.Language, sub.ID)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
