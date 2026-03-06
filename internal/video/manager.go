package video

import (
	"context"
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
	"Kairo/internal/db/dal"
	"Kairo/internal/db/schema"
	"Kairo/internal/deps"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type Manager struct {
	ctx           context.Context
	db            *gorm.DB
	aiService     *ai.Manager
	deps          *deps.Manager
	subtitleQueue chan SubtitleTask
	videoDAL      *dal.VideoDAL
	subtitleDAL   *dal.SubtitleDAL
	categoryDAL   *dal.CategoryDAL
}

func NewManager(ctx context.Context, db *gorm.DB, d *deps.Manager) *Manager {
	m := &Manager{
		ctx:       ctx,
		db:        db,
		aiService: ai.NewManager(ctx),
		deps:      d,
	}
	if db != nil {
		m.videoDAL = dal.NewVideoDAL(db)
		m.subtitleDAL = dal.NewSubtitleDAL(db)
		m.categoryDAL = dal.NewCategoryDAL(db)
	}
	m.InitSubtitleQueue()
	return m
}

func (m *Manager) SaveVideo(v *schema.Video) error {
	if m.videoDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	v.UpdatedAt = time.Now().Unix()
	// Ensure Tags field is updated from TagsList if needed, or caller handles it.
	// Assuming caller sets Tags string for now or we do it here.
	if len(v.TagsList) > 0 {
		bytes, _ := json.Marshal(v.TagsList)
		v.Tags = string(bytes)
	}
	return m.videoDAL.Save(m.ctx, v)
}

func (m *Manager) CreateFromTask(t *schema.Task) error {
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

	v := &schema.Video{
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
		UpdatedAt:  time.Now().Unix(),
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
	if m.videoDAL == nil {
		return false, fmt.Errorf("database not initialized")
	}
	if taskID == "" && filePath == "" {
		return false, nil
	}
	return m.videoDAL.ExistsByTaskOrPath(m.ctx, taskID, filePath)
}

func (m *Manager) GetVideos(filter schema.VideoFilter) ([]*schema.Video, error) {
	if m.videoDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.videoDAL.List(m.ctx, filter.Status, filter.Query)
	if err != nil {
		return nil, err
	}
	// GORM AfterFind hook handles Tags -> TagsList population
	var videos []*schema.Video
	for i := range rows {
		videos = append(videos, &rows[i])
	}
	return videos, nil
}

func (m *Manager) GetVideo(id string) (*schema.Video, error) {
	if m.videoDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	video, err := m.videoDAL.GetByID(m.ctx, id)
	if err != nil {
		return nil, err
	}
	return video, nil
}

func (m *Manager) getCategoryPrompt(categoryID string) (string, error) {
	if m.categoryDAL == nil || strings.TrimSpace(categoryID) == "" {
		return "", nil
	}
	category, err := m.categoryDAL.GetByID(m.ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return category.Prompt, nil
}

func (m *Manager) GetHighlights(videoID string) ([]schema.VideoHighlight, error) {
	if m.videoDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.videoDAL.ListHighlights(m.ctx, videoID)
	if err != nil {
		return []schema.VideoHighlight{}, nil
	}
	return rows, nil
}

// UpdateHighlightFilePath updates the file path for a specific highlight
func (m *Manager) UpdateHighlightFilePath(highlightID, filePath string) error {
	if m.videoDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	return m.videoDAL.UpdateHighlightFilePath(m.ctx, highlightID, filePath)
}

func (m *Manager) DeleteVideo(id string) error {
	if m.videoDAL == nil || m.subtitleDAL == nil {
		return fmt.Errorf("database not initialized")
	}

	subs, _ := m.GetVideoSubtitles(id)
	for _, sub := range subs {
		if strings.TrimSpace(sub.FilePath) == "" {
			continue
		}
		_ = utils.DeleteFile(sub.FilePath)
	}

	return m.videoDAL.DeleteVideoCascade(m.ctx, id)
}

func (m *Manager) UpdateVideoStatus(id, status, summary, evaluation string, tags string, highlights []schema.VideoHighlight) error {
	if m.videoDAL == nil {
		return fmt.Errorf("database not initialized")
	}

	err := m.videoDAL.UpdateStatus(m.ctx, id, status, summary, evaluation, tags)
	if err == nil {
		if status == "completed" && len(highlights) > 0 {
			_ = m.clearHighlightFiles(id)
			records := make([]schema.VideoHighlight, 0, len(highlights))
			now := time.Now().Unix()
			for _, h := range highlights {
				hID := h.ID
				if hID == "" {
					hID = uuid.New().String()
				}
				records = append(records, schema.VideoHighlight{
					ID:          hID,
					VideoID:     id,
					StartTime:   h.StartTime,
					EndTime:     h.EndTime,
					Title:       h.Title,
					Description: h.Description,
					FilePath:    h.FilePath,
					CreatedAt:   now,
					UpdatedAt:   now,
				})
			}
			_ = m.videoDAL.ReplaceHighlights(m.ctx, id, records)
		}

		// Split tags for frontend compatibility
		var tagsList []string
		if tags != "" {
			tagsList = strings.Split(tags, ",")
		} else {
			tagsList = []string{}
		}

		wailsRuntime.EventsEmit(m.ctx, "video:ai_status", map[string]interface{}{
			"id":         id,
			"status":     status,
			"summary":    summary,
			"evaluation": evaluation,
			"tags":       tagsList,
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
	m.UpdateVideoStatus(id, "processing", "", "", "", nil)

	go func() {
		var subtitlesContent string
		var subtitleStats string
		var energyCandidatesText string
		var energyCandidates []energyCandidate
		var subtitleSegments []subtitleSegment
		subtitlePath := ""
		if subs, err := m.GetVideoSubtitles(v.ID); err == nil {
			for _, sub := range subs {
				if schema.SubtitleStatus(sub.Status) == schema.SubtitleStatusSuccess && strings.TrimSpace(sub.FilePath) != "" {
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
				m.UpdateVideoStatus(id, "failed", "AI is disabled in settings", "", "", nil)
				return
			}
			fmt.Printf("AI Analysis failed: %v\n", err)
			m.UpdateVideoStatus(id, "failed", fmt.Sprintf("Error: %v", err), "", "", nil)
			return
		}

		if len(result.Highlights) == 0 && len(energyCandidates) > 0 {
			result.Highlights = buildFallbackHighlights(energyCandidates)
		}
		result.Highlights = normalizeHighlights(result.Highlights, v.Duration, subtitleSegments, energyCandidates)

		// Convert result highlights to model highlights
		var highlights []schema.VideoHighlight
		for _, h := range result.Highlights {
			highlights = append(highlights, schema.VideoHighlight{
				ID:          uuid.New().String(),
				VideoID:     id,
				StartTime:   h.Start,
				EndTime:     h.End,
				Title:       h.Title,
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
	if m.videoDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	rows, err := m.videoDAL.ListHighlights(m.ctx, videoID)
	if err != nil {
		return err
	}
	for _, h := range rows {
		if strings.TrimSpace(h.FilePath) == "" {
			continue
		}
		if err := utils.DeleteFile(h.FilePath); err != nil {
			fmt.Printf("Failed to remove highlight file %s: %v\n", h.FilePath, err)
		}
	}
	return m.videoDAL.DeleteHighlightsByVideoID(m.ctx, videoID)
}

func buildFallbackHighlights(candidates []energyCandidate) []struct {
	Title       string `json:"title"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
} {
	highlights := make([]struct {
		Title       string `json:"title"`
		Start       string `json:"start"`
		End         string `json:"end"`
		Description string `json:"description"`
	}, 0, len(candidates))
	for _, candidate := range candidates {
		highlights = append(highlights, struct {
			Title       string `json:"title"`
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}{
			Title:       "Highlight",
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
	Title       string
	Description string
}

func normalizeHighlights(highlights []struct {
	Title       string `json:"title"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
}, videoDuration float64, segments []subtitleSegment, candidates []energyCandidate) []struct {
	Title       string `json:"title"`
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
			Title:       h.Title,
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
		Title       string `json:"title"`
		Start       string `json:"start"`
		End         string `json:"end"`
		Description string `json:"description"`
	}, 0, len(merged))
	for _, h := range merged {
		final = append(final, struct {
			Title       string `json:"title"`
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}{
			Title:       h.Title,
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
			if items[i].Title != "" && items[i].Title != last.Title {
				last.Title = last.Title + " / " + items[i].Title
			}
			continue
		}
		merged = append(merged, items[i])
	}
	return merged
}

func (m *Manager) ClipHighlights(videoID string, highlights []schema.VideoHighlight) {
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
		safeStart := strings.ReplaceAll(h.StartTime, ":", "-")
		safeEnd := strings.ReplaceAll(h.EndTime, ":", "-")
		outputName := fmt.Sprintf("%s_clip_%s_%s%s", baseName, safeStart, safeEnd, ext)
		outputPath := filepath.Join(dir, outputName)

		args := []string{"-i", v.FilePath, "-ss", h.StartTime, "-to", h.EndTime, "-c", "copy", "-y", outputPath}

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
		"tags":       v.TagsList, // Use TagsList for frontend
		"highlights": updatedHighlights,
	})
}

func (m *Manager) UpdateSubtitle(subtitleID string, content string, language string) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
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
	if err := m.subtitleDAL.Update(m.ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}
