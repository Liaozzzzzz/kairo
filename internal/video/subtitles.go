package video

import (
	"Kairo/internal/ai"
	"Kairo/internal/config"
	"Kairo/internal/models"
	"Kairo/internal/utils"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (m *Manager) GetVideoSubtitles(videoID string) ([]models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.db.Query(
		`SELECT * FROM video_subtitles WHERE video_id = ? ORDER BY created_at ASC`,
		videoID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []models.VideoSubtitle
	for rows.Next() {
		var s models.VideoSubtitle
		if err := rows.Scan(
			&s.ID,
			&s.VideoID,
			&s.FilePath,
			&s.Language,
			&s.Status,
			&s.Source,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			continue
		}
		subs = append(subs, s)
	}
	return subs, nil
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
		_, _ = m.addSubtitleRecord(v.ID, entry.Path, entry.Language, models.SubtitleStatusSuccess, models.SubtitleSourceBuiltin)
	}

	if len(entries) == 0 {
		outputPath, language, asrErr := m.GenerateSubtitlesByASR(v, dir)
		if asrErr != nil {
			return asrErr
		}
		if strings.TrimSpace(outputPath) != "" {
			_, _ = m.addSubtitleRecord(v.ID, outputPath, language, models.SubtitleStatusSuccess, models.SubtitleSourceASR)
			return nil
		}
	}

	return nil
}

func (m *Manager) GenerateSubtitlesByASR(v *models.Video, dir string) (string, string, error) {
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

	ext := filepath.Ext(v.FilePath)
	baseName := strings.TrimSuffix(filepath.Base(v.FilePath), ext)
	language := utils.DetectLanguageFromText(utils.ExtractTextFromVTT(content))
	outputPath := filepath.Join(dir, baseName+".asr."+language+".vtt")
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		log.Printf("[GenerateSubtitlesByASR] error write vtt file: %v", err)
		return "", "", err
	}
	log.Printf("[GenerateSubtitlesByASR] generate subtitles by ASR success, file: %s", outputPath)
	return outputPath, language, nil
}

func (m *Manager) ImportSubtitle(videoID string, filePath string, language string) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	sub := &models.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		FilePath:  filePath,
		Language:  language,
		Status:    models.SubtitleStatusSuccess,
		Source:    models.SubtitleSourceManual,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := m.db.Exec(
		`INSERT INTO video_subtitles (id, video_id, file_path, language, status, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID,
		sub.VideoID,
		sub.FilePath,
		sub.Language,
		sub.Status,
		sub.Source,
		sub.CreatedAt,
		sub.UpdatedAt,
	)

	return sub, err
}

func (m *Manager) addSubtitleRecord(videoID string, filePath string, language string, status models.SubtitleStatus, source models.SubtitleSource) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	var exists int
	err := m.db.QueryRow(
		"SELECT 1 FROM video_subtitles WHERE video_id = ? AND file_path = ? LIMIT 1",
		videoID,
		filePath,
	).Scan(&exists)
	if err == nil && exists == 1 {
		return nil, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	now := time.Now().UnixMilli()
	sub := &models.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		FilePath:  filePath,
		Language:  language,
		Status:    status,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = m.db.Exec(
		`INSERT INTO video_subtitles (id, video_id, file_path, language, status, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID,
		sub.VideoID,
		sub.FilePath,
		sub.Language,
		sub.Status,
		sub.Source,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (m *Manager) TranslateSubtitle(input models.TranslateSubtitleInput) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(input.TargetLanguage) == "" {
		return nil, fmt.Errorf("target language is empty")
	}
	sub, err := m.getSubtitleByID(input.SubtitleID)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubtitleStatusSuccess {
		return nil, fmt.Errorf("subtitle not ready")
	}

	now := time.Now().UnixMilli()
	newSub := &models.VideoSubtitle{
		ID:        uuid.New().String(),
		VideoID:   input.VideoID,
		FilePath:  "",
		Language:  input.TargetLanguage,
		Status:    models.SubtitleStatusPending,
		Source:    models.SubtitleSourceTranslation,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = m.db.Exec(
		`INSERT INTO video_subtitles (id, video_id, file_path, language, status, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		newSub.ID,
		newSub.VideoID,
		newSub.FilePath,
		newSub.Language,
		newSub.Status,
		newSub.Source,
		newSub.CreatedAt,
		newSub.UpdatedAt,
	)

	if err != nil {
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

func (m *Manager) SaveSubtitleContent(videoID string, language string, content string) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is empty")
	}
	video, err := m.GetVideo(videoID)
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
	return m.addSubtitleRecord(videoID, outputPath, language, models.SubtitleStatusSuccess, models.SubtitleSourceManual)
}

func (m *Manager) DeleteSubtitle(id string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	sub, err := m.getSubtitleByID(id)
	if err != nil {
		return err
	}

	if err := utils.DeleteFile(sub.FilePath); err != nil {
		return err
	}

	_, err = m.db.Exec("DELETE FROM video_subtitles WHERE id = ?", id)
	return err
}

func (m *Manager) RegenerateSubtitle(subtitleID string) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sub, err := m.getSubtitleByID(subtitleID)
	if err != nil {
		return nil, err
	}

	if sub.Source != models.SubtitleSourceASR && sub.Source != models.SubtitleSourceTranslation {
		return nil, fmt.Errorf("subtitle source cannot be regenerated")
	}

	sub.Status = models.SubtitleStatusPending
	sub.UpdatedAt = time.Now().UnixMilli()

	_, err = m.db.Exec("UPDATE video_subtitles SET file_path = ?, status = ?, updated_at = ? WHERE id = ?", "", sub.Status, sub.UpdatedAt, sub.ID)
	if err != nil {
		log.Printf("[SubtitleQueue] failed to update subtitle status: %v", err)
		return nil, err
	}
	// delete existing file
	if err := utils.DeleteFile(sub.FilePath); err != nil {
		return nil, err
	}

	subtitleTask := SubtitleTask{
		Type:       SubtitleTaskTypeASR,
		SubtitleID: sub.ID,
		VideoID:    sub.VideoID,
	}
	if sub.Source == models.SubtitleSourceTranslation {
		subtitleTask.Type = SubtitleTaskTypeTranslate
		subtitleTask.TargetLanguage = sub.Language
		bestSource, err := m.findBestSourceSubtitle(sub.VideoID)
		if err != nil || bestSource == nil {
			log.Printf("[SubtitleQueue] failed to find best source subtitle: %v, subtitle: %+v", err, sub)
			m.db.Exec("UPDATE video_subtitles SET status = ? WHERE id = ?", models.SubtitleStatusFailed, sub.ID)
			return nil, err
		}
		subtitleTask.SourceSubtitleID = bestSource.ID
	}

	m.subtitleQueue <- subtitleTask

	return sub, nil
}

func (m *Manager) getSubtitleByID(id string) (*models.VideoSubtitle, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var sub models.VideoSubtitle
	err := m.db.QueryRow(
		`SELECT * FROM video_subtitles WHERE id = ?`,
		id,
	).Scan(
		&sub.ID,
		&sub.VideoID,
		&sub.FilePath,
		&sub.Language,
		&sub.Status,
		&sub.Source,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	return &sub, err
}

func ensureUniqueSubtitlePath(path, targetLanguage, source string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	outputPath := filepath.Join(dir, base+"."+source+"."+targetLanguage+".vtt")
	if _, err := os.Stat(outputPath); err == nil {
		outputPath = filepath.Join(dir, base+"."+uuid.NewString()+"."+source+"."+targetLanguage+".vtt")
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
		b.WriteString(formatSubtitleTimestamp(seg.Start))
		b.WriteString(" --> ")
		b.WriteString(formatSubtitleTimestamp(seg.End))
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(translations[i]))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String()) + "\n"
}
