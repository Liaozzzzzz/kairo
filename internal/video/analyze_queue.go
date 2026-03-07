package video

import (
	"fmt"
	"os"
	"strings"

	"Kairo/internal/config"
	"Kairo/internal/db/schema"
)

func (m *Manager) InitAnalyzeQueue() {
	m.analysisQueue = make(chan string, 100)
	go m.resetProcessingVideoStatus()
	go m.processAnalyzeQueue()
	go m.enqueueUnanalyzedVideos()
}

func (m *Manager) resetProcessingVideoStatus() {
	if m.videoDAL == nil {
		return
	}
	_ = m.videoDAL.UpdateStatusByStatus(m.ctx, "processing", "none")
}

func (m *Manager) processAnalyzeQueue() {
	for videoID := range m.analysisQueue {
		m.handleAnalyzeTask(videoID)
	}
}

func (m *Manager) handleAnalyzeTask(videoID string) {
	video, err := m.GetVideoById(videoID)
	if err != nil {
		return
	}
	status := strings.TrimSpace(video.Status)
	if status != "" && status != "none" {
		return
	}
	if _, err := m.getReadySubtitlePath(videoID); err != nil {
		return
	}
	_ = m.AnalyzeVideo(videoID)
}

func (m *Manager) enqueueUnanalyzedVideos() {
	if !config.GetSettings().AI.Enabled {
		return
	}
	if m.videoDAL == nil {
		return
	}
	videos, err := m.videoDAL.ListUnanalyzed(m.ctx, 50)
	if err != nil {
		return
	}
	for _, video := range videos {
		if _, err := m.getReadySubtitlePath(video.ID); err != nil {
			continue
		}
		m.enqueueAnalyze(video.ID)
	}
}

func (m *Manager) enqueueAnalyze(videoID string) {
	if m.analysisQueue == nil {
		return
	}
	m.analysisQueue <- videoID
}

func (m *Manager) getReadySubtitlePath(videoID string) (string, error) {
	if m.subtitleDAL == nil {
		return "", fmt.Errorf("database not initialized")
	}
	if best, err := m.findBestSourceSubtitle(videoID); err == nil && best != nil {
		path := strings.TrimSpace(best.FilePath)
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	subs, err := m.GetVideoSubtitles(videoID)
	if err != nil {
		return "", err
	}
	for _, sub := range subs {
		if schema.SubtitleStatus(sub.Status) != schema.SubtitleStatusSuccess {
			continue
		}
		path := strings.TrimSpace(sub.FilePath)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("subtitle not ready")
}
