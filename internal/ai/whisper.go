package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Kairo/internal/config"
)

func (m *Manager) TranscribeWhisper(filePath string, responseFormat string) (string, error) {
	if responseFormat == "" {
		responseFormat = "vtt"
	}
	data, err := m.transcribeWhisperRaw(filePath, responseFormat)
	if err != nil {
		return "", err
	}
	if responseFormat != "vtt" {
		return string(data), nil
	}
	if hasVttTimestamps(string(data)) {
		return string(data), nil
	}
	if vtt, ok := buildVttFromVerboseJSON(data); ok {
		return vtt, nil
	}
	verboseData, err := m.transcribeWhisperRaw(filePath, "verbose_json")
	if err == nil {
		if vtt, ok := buildVttFromVerboseJSON(verboseData); ok {
			return vtt, nil
		}
	}
	srtData, err := m.transcribeWhisperRaw(filePath, "srt")
	if err == nil {
		if vtt, ok := convertSrtToVtt(string(srtData)); ok {
			return vtt, nil
		}
	}
	return string(data), nil
}

type whisperVerboseResponse struct {
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

func buildVttFromVerboseJSON(data []byte) (string, bool) {
	var resp whisperVerboseResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", false
	}
	if len(resp.Segments) == 0 {
		return "", false
	}
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, seg := range resp.Segments {
		start := formatVttTimestamp(seg.Start)
		end := formatVttTimestamp(seg.End)
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, start, end, text)
	}
	return b.String(), true
}

func formatVttTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	totalMillis := int64(seconds*1000 + 0.5)
	hours := totalMillis / 3600000
	minutes := (totalMillis % 3600000) / 60000
	secs := (totalMillis % 60000) / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, secs, millis)
}

func hasVttTimestamps(data string) bool {
	return strings.Contains(data, "-->")
}

func convertSrtToVtt(srt string) (string, bool) {
	s := strings.ReplaceAll(srt, "\r\n", "\n")
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	lines := strings.Split(s, "\n")
	hasTimestamp := false
	for i, line := range lines {
		if strings.Contains(line, "-->") {
			hasTimestamp = true
			lines[i] = strings.ReplaceAll(line, ",", ".")
		}
	}
	if !hasTimestamp {
		return "", false
	}
	if strings.HasPrefix(lines[0], "WEBVTT") {
		return strings.Join(lines, "\n"), true
	}
	return "WEBVTT\n\n" + strings.Join(lines, "\n"), true
}

func (m *Manager) transcribeWhisperRaw(filePath string, responseFormat string) ([]byte, error) {
	cfg := config.GetSettings().WhisperAI
	if !cfg.Enabled {
		return nil, ErrWhisperDisabled
	}

	model := cfg.ModelName
	if model == "" {
		model = "whisper-1"
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(filePart, f); err != nil {
		return nil, err
	}
	if err := writer.WriteField("model", model); err != nil {
		return nil, err
	}
	if err := writer.WriteField("response_format", responseFormat); err != nil {
		return nil, err
	}
	if cfg.Prompt != "" {
		if err := writer.WriteField("prompt", cfg.Prompt); err != nil {
			return nil, err
		}
	}
	if cfg.Language != "" {
		if err := writer.WriteField("language", cfg.Language); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/audio/transcriptions", strings.TrimRight(cfg.BaseURL, "/"))
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cfg.APIKey != "" {
		switch strings.ToLower(cfg.Provider) {
		case "openai", "local", "siliconflow":
			req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		default:
			req.Header.Set("api-key", cfg.APIKey)
		}
	}

	client := *m.client
	client.Timeout = 5 * time.Minute
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
