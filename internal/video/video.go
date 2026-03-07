package video

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/config"
	"Kairo/internal/db/schema"
	"Kairo/internal/utils"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

func (m *Manager) ListVideos(filter schema.VideoFilter) ([]schema.Video, error) {
	return m.videoDAL.List(m.ctx, filter.Status, filter.Query)
}

func (m *Manager) GetVideoById(id string) (*schema.Video, error) {
	if m.videoDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	video, err := m.videoDAL.GetByID(m.ctx, id)
	if err != nil {
		return nil, err
	}
	return video, nil
}

func (m *Manager) DeleteVideo(id string) error {
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
	if err := m.videoDAL.UpdateStatus(m.ctx, id, status, summary, evaluation, tags); err != nil {
		return err
	}

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
			_ = m.highlightDAL.Replace(m.ctx, id, records)
		}

		wailsRuntime.EventsEmit(m.ctx, "video:ai_status", map[string]interface{}{
			"id":         id,
			"status":     status,
			"summary":    summary,
			"evaluation": evaluation,
			"tags":       strings.Split(tags, ","),
			"highlights": highlights,
		})
	}

	return nil
}

func (m *Manager) AnalyzeVideo(id string) error {
	if !config.GetSettings().AI.Enabled {
		return nil
	}

	v, err := m.GetVideoById(id)
	if err != nil {
		return err
	}

	subtitlePath, err := m.getReadySubtitlePath(v.ID)
	if err != nil {
		return err
	}

	// Set status to analyzing
	// If re-analyzing, we clear previous results but keep metadata
	m.UpdateVideoStatus(id, "processing", "", "", "", nil)

	go func(subtitlePath string) {
		var subtitlesContent string
		var subtitleStats string
		var energyCandidatesText string
		var energyCandidates []energyCandidate
		var subtitleSegments []subtitleSegment
		if segments, err := parseSubtitleFile(subtitlePath); err == nil && len(segments) > 0 {
			subtitleSegments = segments
			subtitlesContent = buildSubtitleText(segments)
			subtitleStats, energyCandidates = buildSubtitleAnalysis(segments, v.Duration)
			energyCandidatesText = formatEnergyCandidates(energyCandidates)
		} else if content, readErr := os.ReadFile(subtitlePath); readErr == nil {
			subtitlesContent = string(content)
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
	}(subtitlePath)

	return nil
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
