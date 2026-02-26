package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"Kairo/internal/config"
)

func (m *Manager) callOpenAI(cfg config.AIConfig, prompt string, jsonMode bool) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimRight(cfg.BaseURL, "/"))

	reqBody := map[string]interface{}{
		"model": cfg.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if jsonMode {
		reqBody["response_format"] = map[string]string{"type": "json_object"}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	preview := prompt
	if len(preview) > 5000 {
		preview = preview[:5000] + "..."
	}
	log.Printf("[AI Request] URL: %s\nModel: %s\nPrompt: %s\n", url, cfg.ModelName, preview)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := *m.client
	client.Timeout = 5 * time.Minute

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	content := result.Choices[0].Message.Content
	log.Printf("[AI Response] Content: %s\n", content)
	return content, nil
}

func (m *Manager) callAnthropic(cfg config.AIConfig, prompt string) (string, error) {
	return "", fmt.Errorf("Anthropic provider not fully implemented yet")
}

func sanitizeJSONContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
		if strings.HasSuffix(trimmed, "```") {
			trimmed = strings.TrimSuffix(trimmed, "```")
			trimmed = strings.TrimSpace(trimmed)
		}
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return trimmed
}
