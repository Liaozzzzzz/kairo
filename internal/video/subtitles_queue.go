package video

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"Kairo/internal/models"
)

type SubtitleTaskType int

const (
	SubtitleTaskTypeASR SubtitleTaskType = iota
	SubtitleTaskTypeTranslate
)

type SubtitleTask struct {
	Type             SubtitleTaskType
	SubtitleID       string // The ID of the subtitle being generated (the target)
	VideoID          string
	SourceSubtitleID string // For translation: The ID of the source subtitle
	TargetLanguage   string // For translation
}

func (m *Manager) InitSubtitleQueue() {
	m.subtitleQueue = make(chan SubtitleTask, 100)
	go m.restorePendingSubtitles()
	go m.processSubtitleQueue()
}

func (m *Manager) restorePendingSubtitles() {
	// 1. Reset Generating -> Pending
	log.Println("[SubtitleQueue] Resetting generating subtitles to pending...")
	_, err := m.db.Exec("UPDATE video_subtitles SET status = ? WHERE status = ?", models.SubtitleStatusPending, models.SubtitleStatusGenerating)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to reset generating subtitles: %v", err)
	}

	// 2. Load Pending tasks
	rows, err := m.db.Query("SELECT id, video_id, language, status, source FROM video_subtitles WHERE status = ?", models.SubtitleStatusPending)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to query pending subtitles: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var subtitle models.VideoSubtitle
		if err := rows.Scan(&subtitle.ID,
			&subtitle.VideoID,
			&subtitle.Language,
			&subtitle.Status,
			&subtitle.Source); err != nil {
			log.Printf("[SubtitleQueue] failed to scan pending subtitle: %v", err)
			continue
		}

		log.Printf("[SubtitleQueue] Restored pending subtitle: %+v", subtitle)

		task := SubtitleTask{
			SubtitleID: subtitle.ID,
			VideoID:    subtitle.VideoID,
		}

		if subtitle.Source == models.SubtitleSourceTranslation {
			task.Type = SubtitleTaskTypeTranslate
			task.TargetLanguage = subtitle.Language
			bestSource, err := m.findBestSourceSubtitle(subtitle.VideoID)

			if err != nil || bestSource == nil {
				log.Printf("[SubtitleQueue] failed to find best source subtitle: %v, subtitle: %+v", err, subtitle)
				m.db.Exec("UPDATE video_subtitles SET status = ? WHERE id = ?", models.SubtitleStatusFailed, subtitle.ID)
				continue
			}
			task.SourceSubtitleID = bestSource.ID
			m.subtitleQueue <- task
			count++
		} else if subtitle.Source == models.SubtitleSourceASR {
			task.Type = SubtitleTaskTypeASR
			m.subtitleQueue <- task
			count++
		}
	}
	log.Printf("[SubtitleQueue] Restored %d pending subtitles", count)
}

func (m *Manager) findBestSourceSubtitle(videoID string) (*models.VideoSubtitle, error) {
	// Prefer ASR, then Builtin
	rows, err := m.db.Query("SELECT id, source FROM video_subtitles WHERE video_id = ? AND status = ?", videoID, models.SubtitleStatusSuccess)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bestID string
	var bestScore int = -1
	for rows.Next() {
		var id string
		var source models.SubtitleSource
		if err := rows.Scan(&id, &source); err != nil {
			continue
		}
		score := 0
		if source == models.SubtitleSourceASR {
			score = 10
		} else if source == models.SubtitleSourceBuiltin {
			score = 5
		} else if source == models.SubtitleSourceManual {
			score = 8
		}

		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}

	if bestID != "" {
		return m.getSubtitleByID(bestID)
	}
	return nil, fmt.Errorf("no suitable source found")
}

func (m *Manager) processSubtitleQueue() {
	for task := range m.subtitleQueue {
		m.handleSubtitleTask(task)
	}
}

func (m *Manager) handleSubtitleTask(task SubtitleTask) {
	log.Printf("[SubtitleQueue] processing task: %+v", task)

	// Update status to Generating
	_, err := m.db.Exec("UPDATE video_subtitles SET status = ? WHERE id = ?", models.SubtitleStatusGenerating, task.SubtitleID)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to update status to generating: %v", err)
		return
	}

	var resultErr error

	if task.Type == SubtitleTaskTypeASR {
		resultErr = m.processASRTask(task)
	} else if task.Type == SubtitleTaskTypeTranslate {
		resultErr = m.processTranslateTask(task)
	}

	status := models.SubtitleStatusSuccess
	if resultErr != nil {
		status = models.SubtitleStatusFailed
		log.Printf("[SubtitleQueue] task failed: %v", resultErr)
	}

	_, err = m.db.Exec("UPDATE video_subtitles SET status = ?, updated_at = ? WHERE id = ?", status, time.Now().UnixMilli(), task.SubtitleID)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to update status to %v: %v", status, err)
	}
}

func (m *Manager) processASRTask(task SubtitleTask) error {
	video, err := m.GetVideo(task.VideoID)
	if err != nil {
		return err
	}
	dir := filepath.Dir(video.FilePath)
	outputPath, language, err := m.GenerateSubtitlesByASR(video, dir)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(
		`UPDATE video_subtitles SET file_path = ?, language = ? WHERE id = ?`,
		outputPath,
		language,
		task.SubtitleID,
	)
	return err
}

func (m *Manager) processTranslateTask(task SubtitleTask) error {
	// 1. Get source subtitle
	sourceSub, err := m.getSubtitleByID(task.SourceSubtitleID)
	if err != nil {
		return fmt.Errorf("failed to get source subtitle: %v", err)
	}

	// 2. Parse source
	segments, err := parseSubtitleFile(sourceSub.FilePath)
	if err != nil {
		return fmt.Errorf("failed to parse source subtitle: %v", err)
	}

	// 3. Translate
	var texts []string
	for _, seg := range segments {
		texts = append(texts, seg.Text)
	}
	translations, err := m.aiService.TranslateSegments(task.TargetLanguage, texts)
	if err != nil {
		return fmt.Errorf("translation failed: %v", err)
	}
	content := buildTranslatedVTT(segments, translations)

	// 4. Save file
	video, err := m.GetVideo(task.VideoID)
	if err != nil {
		return err
	}

	outputPath := ensureUniqueSubtitlePath(video.FilePath, task.TargetLanguage, "trans")
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// 5. Update DB (file_path)
	_, err = m.db.Exec("UPDATE video_subtitles SET file_path = ? WHERE id = ?", outputPath, task.SubtitleID)
	return err
}
