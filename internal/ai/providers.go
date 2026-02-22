package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"Kairo/internal/config"
)

func (m *Manager) callOpenAI(cfg config.AIConfig, prompt string) (*AnalysisResult, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimRight(cfg.BaseURL, "/"))

	reqBody := map[string]interface{}{
		"model": cfg.ModelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	preview := prompt
	if len(preview) > 5000 {
		preview = preview[:5000] + "..."
	}
	fmt.Printf("[AI Request] URL: %s\nModel: %s\nPrompt: %s\n", url, cfg.ModelName, preview)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := result.Choices[0].Message.Content

	fmt.Printf("[AI Response] Content: %s\n", content)

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		fmt.Printf("[AI Response Error] Failed to parse JSON: %v\n", err)
		analysis.Summary = content
		analysis.Tags = []string{}
		analysis.Highlights = []struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Description string `json:"description"`
		}{}
		analysis.Evaluation = "Failed to parse JSON response"
	}

	return &analysis, nil
}

func (m *Manager) callAnthropic(cfg config.AIConfig, prompt string) (*AnalysisResult, error) {
	return nil, fmt.Errorf("Anthropic provider not fully implemented yet")
}
