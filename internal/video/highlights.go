package video

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"Kairo/internal/db/schema"
	"Kairo/internal/utils"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (m *Manager) GetHighlights(videoID string) ([]schema.VideoHighlight, error) {
	if m.highlightDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.highlightDAL.ListByVideoID(m.ctx, videoID)
	if err != nil {
		return []schema.VideoHighlight{}, nil
	}
	return rows, nil
}

// UpdateHighlightFilePath updates the file path for a specific highlight
func (m *Manager) UpdateHighlightFilePath(highlightID, filePath string) error {
	if m.highlightDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	return m.highlightDAL.UpdateFilePath(m.ctx, highlightID, filePath)
}

func (m *Manager) clearHighlightFiles(videoID string) error {
	if m.highlightDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	rows, err := m.highlightDAL.ListByVideoID(m.ctx, videoID)
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
	return m.highlightDAL.DeleteByVideoID(m.ctx, videoID)
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
			Start:       formatTimestamp(candidate.Start, false),
			End:         formatTimestamp(candidate.End, false),
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
		startSec, errStart := parseTimestampToSeconds(h.Start)
		endSec, errEnd := parseTimestampToSeconds(h.End)
		if errStart != nil || errEnd != nil || endSec <= startSec {
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
			Start:       formatTimestamp(h.Start, false),
			End:         formatTimestamp(h.End, false),
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
	v, err := m.GetVideoById(videoID)
	if err != nil {
		fmt.Printf("Failed to get video for clipping: %v\n", err)
		return
	}

	ffmpegPath, err := m.deps.GetFFmpegPath()
	if err != nil {
		fmt.Printf("Failed to get ffmpeg: %v\n", err)
		return
	}

	pathInfo := buildVideoPathInfo(v.FilePath)

	for _, h := range highlights {
		safeStart := strings.ReplaceAll(h.StartTime, ":", "-")
		safeEnd := strings.ReplaceAll(h.EndTime, ":", "-")
		outputName := fmt.Sprintf("%s_clip_%s_%s%s", pathInfo.BaseName, safeStart, safeEnd, pathInfo.Ext)
		outputPath := filepath.Join(pathInfo.Dir, outputName)

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
