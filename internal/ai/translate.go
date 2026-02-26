package ai

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"Kairo/internal/config"
)

//go:embed prompts/translate.txt
var defaultTranslatePrompt string

func (m *Manager) TranslateSegments(targetLanguage string, segments []string) ([]string, error) {
	settings := config.GetSettings()
	if !settings.TranslateAI.Enabled {
		return nil, ErrAIDisabled
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("no segments to translate")
	}

	// Batch processing configuration
	batchSize := 50
	chunks := chunkSlice(segments, batchSize)
	results := make([][]string, len(chunks))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var finalErr error

	// Limit concurrency to avoid rate limits
	maxConcurrency := 5
	sem := make(chan struct{}, maxConcurrency)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, segs []string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check if we should abort due to previous error
			mu.Lock()
			if finalErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			translated, err := m.translateBatch(settings, targetLanguage, segs)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				// Record the first error and stop processing
				if finalErr == nil {
					finalErr = err
				}
				return
			}
			results[idx] = translated
		}(i, chunk)
	}

	wg.Wait()

	if finalErr != nil {
		return nil, finalErr
	}

	// Flatten results
	var finalTranslations []string
	for _, res := range results {
		finalTranslations = append(finalTranslations, res...)
	}

	return finalTranslations, nil
}

func (m *Manager) translateBatch(settings config.AppSettings, targetLanguage string, segments []string) ([]string, error) {
	payload, _ := json.Marshal(segments)

	prompt := defaultTranslatePrompt
	prompt = strings.ReplaceAll(prompt, "{{target_language}}", targetLanguage)
	prompt = strings.ReplaceAll(prompt, "{{segments}}", string(payload))
	if settings.TranslateAI.Prompt != "" {
		prompt = prompt + "\n\n" + settings.TranslateAI.Prompt
	}

	var content string
	var err error
	switch settings.TranslateAI.Provider {
	case "openai", "local", "custom", "deepseek", "siliconflow":
		content, err = m.callOpenAI(settings.TranslateAI, prompt, true)
	case "anthropic":
		content, err = m.callAnthropic(settings.TranslateAI, prompt)
	default:
		content, err = m.callOpenAI(settings.TranslateAI, prompt, true)
	}
	if err != nil {
		log.Printf("[Translate] Error calling AI provider: %v", err)
		return nil, err
	}

	content = sanitizeJSONContent(content)
	var result struct {
		Translations []string `json:"translations"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("[Translate] Error unmarshalling translation response: %v", err)
		return nil, err
	}
	return result.Translations, nil
}

func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
