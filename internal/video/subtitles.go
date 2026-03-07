package video

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Kairo/internal/ai"
	"Kairo/internal/config"
	"Kairo/internal/db/schema"
	"Kairo/internal/utils"

	"github.com/google/uuid"
)

func (m *Manager) GetVideoSubtitles(videoID string) ([]schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return m.subtitleDAL.ListByVideoID(m.ctx, videoID)
}

func (m *Manager) FetchSubtitles(id string) error {
	v, err := m.GetVideoById(id)
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
	outputTemplate := buildOutputTemplate(v.FilePath)
	ffmpegPath, _ := m.deps.GetFFmpegPath()

	args := []string{
		"--skip-download",
		"--write-subs",
		"--write-auto-subs",
		"--ffmpeg-location", filepath.Dir(ffmpegPath),
		"-o", outputTemplate,
	}

	// if bili video, add --sub-langs "Hans,-danmaku"
	if strings.Contains(v.URL, "bilibili") {
		args = append(args, "--sub-langs", "zh-Hans,-danmaku")
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
	cmd := utils.CreateCommandContext(m.ctx, ytDlpPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[FetchSubtitles] error fetch subtitles: %v, output: %s", err, string(output))
	}

	entries := extractSubtitleEntriesFromOutput(string(output))
	for _, entry := range entries {
		_, _ = m.addSubtitleRecord(v.ID, entry.Path, entry.Language, schema.SubtitleStatusSuccess, schema.SubtitleSourceBuiltin)
	}

	if len(entries) == 0 {
		outputPath, language, asrErr := m.GenerateSubtitlesByASR(v)
		if asrErr != nil {
			return asrErr
		}

		if strings.TrimSpace(outputPath) == "" {
			return fmt.Errorf("asr subtitles failed")
		}

		if _, err := m.addSubtitleRecord(v.ID, outputPath, language, schema.SubtitleStatusSuccess, schema.SubtitleSourceASR); err != nil {
			return err
		}

		m.enqueueAnalyze(v.ID)
		return nil
	}

	if len(entries) > 0 {
		m.enqueueAnalyze(v.ID)
	}

	return err
}

func (m *Manager) GenerateSubtitlesByASR(v *schema.Video) (string, string, error) {
	if !config.GetSettings().WhisperAI.Enabled {
		log.Printf("[GenerateSubtitlesByASR] Whisper disabled, skip generating subtitles")
		return "", "", nil
	}

	log.Printf("[GenerateSubtitlesByASR] start generate subtitles by ASR for video %s,", v.ID)
	inputPath := v.FilePath
	tempPath := ""
	if strings.ToLower(filepath.Ext(v.FilePath)) != ".mp3" {
		ffmpegPath, err := m.deps.GetFFmpegPath()
		if err != nil {
			log.Printf("[GenerateSubtitlesByASR] error get ffmpeg path: %v", err)
			return "", "", fmt.Errorf("asr failed: %v", err)
		}
		tempDir := filepath.Dir(v.FilePath)
		tempFile, err := os.CreateTemp(tempDir, "whisper-*.mp3")
		if err != nil {
			log.Printf("[GenerateSubtitlesByASR] error create temp file: %v", err)
			return "", "", err
		}
		tempPath = tempFile.Name()
		if err := tempFile.Close(); err != nil {
			log.Printf("[GenerateSubtitlesByASR] error close temp file: %v", err)
			return "", "", err
		}
		args := []string{"-i", v.FilePath, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "libmp3lame", "-y", tempPath}
		cmd := utils.CreateCommand(ffmpegPath, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[GenerateSubtitlesByASR] error convert to mp3: %v, output: %s", err, string(output))
			_ = os.Remove(tempPath)
			return "", "", fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
		}
		inputPath = tempPath
	}
	if tempPath != "" {
		defer os.Remove(tempPath)
	}
	content, err := m.aiService.TranscribeWhisper(inputPath)
	if err != nil {
		log.Printf("[GenerateSubtitlesByASR] error transcribe whisper: %v", err)
		if errors.Is(err, ai.ErrWhisperDisabled) {
			log.Printf("[GenerateSubtitlesByASR] Whisper disabled, skip generating subtitles")
			return "", "", nil
		}
		return "", "", fmt.Errorf("asr failed: %v", err)
	}

	pathInfo := buildVideoPathInfo(v.FilePath)
	language := utils.DetectLanguageFromText(utils.ExtractTextFromVTT(content))
	outputPath := filepath.Join(pathInfo.Dir, pathInfo.BaseName+".asr."+language+".vtt")
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		log.Printf("[GenerateSubtitlesByASR] error write vtt file: %v", err)
		return "", "", err
	}
	log.Printf("[GenerateSubtitlesByASR] generate subtitles by ASR success, file: %s", outputPath)
	return outputPath, language, nil
}

func (m *Manager) ImportSubtitle(videoID string, filePath string, language string) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	sub := &schema.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		FilePath:  filePath,
		Language:  language,
		Status:    schema.SubtitleStatusSuccess,
		Source:    schema.SubtitleSourceManual,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := m.subtitleDAL.Create(m.ctx, sub)
	return sub, err
}

func (m *Manager) addSubtitleRecord(videoID string, filePath string, language string, status schema.SubtitleStatus, source schema.SubtitleSource) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	exists, err := m.subtitleDAL.ExistsByVideoAndPath(m.ctx, videoID, filePath)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, nil
	}
	now := time.Now().UnixMilli()
	sub := &schema.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		FilePath:  filePath,
		Language:  language,
		Status:    status,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := m.subtitleDAL.Create(m.ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (m *Manager) TranslateSubtitle(input schema.TranslateSubtitleInput) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(input.TargetLanguage) == "" {
		return nil, fmt.Errorf("target language is empty")
	}
	sub, err := m.getSubtitleByID(input.SubtitleID)
	if err != nil {
		return nil, err
	}
	if schema.SubtitleStatus(sub.Status) != schema.SubtitleStatusSuccess {
		return nil, fmt.Errorf("subtitle not ready")
	}

	now := time.Now().UnixMilli()
	newSub := &schema.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   input.VideoID,
		FilePath:  "",
		Language:  input.TargetLanguage,
		Status:    schema.SubtitleStatusPending,
		Source:    schema.SubtitleSourceTranslation,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := m.subtitleDAL.Create(m.ctx, newSub); err != nil {
		return nil, err
	}

	m.subtitleQueue <- SubtitleTask{
		Type:             SubtitleTaskTypeTranslate,
		SubtitleID:       newSub.ID,
		VideoID:          newSub.VideoID,
		SourceSubtitleID: input.SubtitleID,
		TargetLanguage:   input.TargetLanguage,
	}

	return newSub, nil
}

func (m *Manager) SaveSubtitleContent(videoID string, language string, content string) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is empty")
	}
	video, err := m.GetVideoById(videoID)
	if err != nil {
		return nil, err
	}
	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = "unknown"
	}

	outputPath := ensureUniqueSubtitlePath(video.FilePath, lang, "manual")
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return m.addSubtitleRecord(videoID, outputPath, language, schema.SubtitleStatusSuccess, schema.SubtitleSourceManual)
}

func (m *Manager) DeleteSubtitle(id string) error {
	if m.subtitleDAL == nil {
		return fmt.Errorf("database not initialized")
	}
	sub, err := m.getSubtitleByID(id)
	if err != nil {
		return err
	}

	if err := utils.DeleteFile(sub.FilePath); err != nil {
		return err
	}

	return m.subtitleDAL.DeleteByID(m.ctx, id)
}

func (m *Manager) RegenerateSubtitle(subtitleID string) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sub, err := m.getSubtitleByID(subtitleID)
	if err != nil {
		return nil, err
	}

	if schema.SubtitleSource(sub.Source) != schema.SubtitleSourceASR && schema.SubtitleSource(sub.Source) != schema.SubtitleSourceTranslation {
		return nil, fmt.Errorf("subtitle source cannot be regenerated")
	}

	oldPath := sub.FilePath
	sub.Status = schema.SubtitleStatusPending
	sub.UpdatedAt = time.Now().UnixMilli()
	sub.FilePath = ""
	err = m.subtitleDAL.Update(m.ctx, sub)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to update subtitle status: %v", err)
		return nil, err
	}
	// delete existing file
	if err := utils.DeleteFile(oldPath); err != nil {
		return nil, err
	}

	subtitleTask := SubtitleTask{
		Type:       SubtitleTaskTypeASR,
		SubtitleID: sub.ID,
		VideoID:    sub.VideoID,
	}
	if schema.SubtitleSource(sub.Source) == schema.SubtitleSourceTranslation {
		subtitleTask.Type = SubtitleTaskTypeTranslate
		subtitleTask.TargetLanguage = sub.Language
		bestSource, err := m.findBestSourceSubtitle(sub.VideoID)
		if err != nil || bestSource == nil {
			log.Printf("[SubtitleQueue] failed to find best source subtitle: %v, subtitle: %+v", err, sub)
			sub.Status = schema.SubtitleStatusFailed
			sub.UpdatedAt = time.Now().UnixMilli()
			_ = m.subtitleDAL.Update(m.ctx, sub)
			return nil, err
		}
		subtitleTask.SourceSubtitleID = bestSource.ID
	}

	m.subtitleQueue <- subtitleTask

	return sub, nil
}

func (m *Manager) getSubtitleByID(id string) (*schema.VideoSubtitle, error) {
	if m.subtitleDAL == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sub, err := m.subtitleDAL.GetByID(m.ctx, id)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func ensureUniqueSubtitlePath(path, targetLanguage, source string) string {
	pathInfo := buildVideoPathInfo(path)
	outputPath := filepath.Join(pathInfo.Dir, pathInfo.BaseName+"."+source+"."+targetLanguage+".vtt")
	if _, err := os.Stat(outputPath); err == nil {
		outputPath = filepath.Join(pathInfo.Dir, pathInfo.BaseName+"."+uuid.NewString()+"."+source+"."+targetLanguage+".vtt")
	}
	return outputPath
}

func buildTranslatedVTT(segments []subtitleSegment, translations []string) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, seg := range segments {
		if i >= len(translations) {
			break
		}
		b.WriteString(formatTimestamp(seg.Start, true))
		b.WriteString(" --> ")
		b.WriteString(formatTimestamp(seg.End, true))
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(translations[i]))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String()) + "\n"
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
		video, err := m.GetVideoById(sub.VideoID)
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
