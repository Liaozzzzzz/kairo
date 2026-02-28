package video

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"Kairo/internal/db/schema"
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
	if m.subtitleDAL == nil {
		return
	}
	if err := m.subtitleDAL.UpdateStatusByStatus(m.ctx, int(schema.SubtitleStatusGenerating), int(schema.SubtitleStatusPending)); err != nil {
		log.Printf("[SubtitleQueue] failed to reset generating subtitles: %v", err)
	}

	// 2. Load Pending tasks
	rows, err := m.subtitleDAL.ListByStatus(m.ctx, int(schema.SubtitleStatusPending))
	if err != nil {
		log.Printf("[SubtitleQueue] failed to query pending subtitles: %v", err)
		return
	}

	count := 0
	for _, subtitle := range rows {
		log.Printf("[SubtitleQueue] Restored pending subtitle: %+v", subtitle)

		task := SubtitleTask{
			SubtitleID: subtitle.ID,
			VideoID:    subtitle.VideoID,
		}

		if schema.SubtitleSource(subtitle.Source) == schema.SubtitleSourceTranslation {
			task.Type = SubtitleTaskTypeTranslate
			task.TargetLanguage = subtitle.Language
			bestSource, err := m.findBestSourceSubtitle(subtitle.VideoID)

			if err != nil || bestSource == nil {
				log.Printf("[SubtitleQueue] failed to find best source subtitle: %v, subtitle: %+v", err, subtitle)
				_ = m.subtitleDAL.UpdateStatus(m.ctx, subtitle.ID, int(schema.SubtitleStatusFailed))
				continue
			}
			task.SourceSubtitleID = bestSource.ID
			m.subtitleQueue <- task
			count++
		} else if schema.SubtitleSource(subtitle.Source) == schema.SubtitleSourceASR {
			task.Type = SubtitleTaskTypeASR
			m.subtitleQueue <- task
			count++
		}
	}
	log.Printf("[SubtitleQueue] Restored %d pending subtitles", count)
}

func (m *Manager) findBestSourceSubtitle(videoID string) (*schema.VideoSubtitle, error) {
	// Prefer ASR, then Builtin
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.subtitleDAL.ListByVideoAndStatus(m.ctx, videoID, int(schema.SubtitleStatusSuccess))
	if err != nil {
		return nil, err
	}

	var bestID string
	var bestScore int = -1
	for _, s := range rows {
		id := s.ID
		source := schema.SubtitleSource(s.Source)
		score := 0
		if source == schema.SubtitleSourceASR {
			score = 10
		} else if source == schema.SubtitleSourceBuiltin {
			score = 5
		} else if source == schema.SubtitleSourceManual {
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
	if m.subtitleDAL == nil {
		return
	}
	if err := m.subtitleDAL.UpdateStatus(m.ctx, task.SubtitleID, int(schema.SubtitleStatusGenerating)); err != nil {
		log.Printf("[SubtitleQueue] failed to update status to generating: %v", err)
		return
	}

	var resultErr error

	if task.Type == SubtitleTaskTypeASR {
		resultErr = m.processASRTask(task)
	} else if task.Type == SubtitleTaskTypeTranslate {
		resultErr = m.processTranslateTask(task)
	}

	status := schema.SubtitleStatusSuccess
	if resultErr != nil {
		status = schema.SubtitleStatusFailed
		log.Printf("[SubtitleQueue] task failed: %v", resultErr)
	}
	if err := m.subtitleDAL.UpdateStatus(m.ctx, task.SubtitleID, int(status)); err != nil {
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
	sub, err := m.getSubtitleByID(task.SubtitleID)
	if err != nil {
		return err
	}
	sub.FilePath = outputPath
	sub.Language = language
	sub.UpdatedAt = time.Now().UnixMilli()
	return m.subtitleDAL.Update(m.ctx, sub)
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
	sub, err := m.getSubtitleByID(task.SubtitleID)
	if err != nil {
		return err
	}
	sub.FilePath = outputPath
	sub.UpdatedAt = time.Now().UnixMilli()
	return m.subtitleDAL.Update(m.ctx, sub)
}
