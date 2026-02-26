package ai

import (
	_ "embed"
	"encoding/json"
	"log"
	"strings"

	"Kairo/internal/config"
)

//go:embed prompts/analysis.txt
var defaultAIPrompt string

type VideoMetadata struct {
	Title            string
	Description      string
	Subtitles        string
	SubtitleStats    string
	EnergyCandidates string
	Uploader         string
	Duration         string
	Resolution       string
	Format           string
	Size             string
	Date             string
}

type AnalysisResult struct {
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Evaluation string   `json:"evaluation"`
	Highlights []struct {
		Start       string `json:"start"`
		End         string `json:"end"`
		Description string `json:"description"`
	} `json:"highlights"`
}

func (m *Manager) Analyze(meta VideoMetadata) (*AnalysisResult, error) {
	settings := config.GetSettings()
	if !settings.AI.Enabled {
		return nil, ErrAIDisabled
	}

	prompt := defaultAIPrompt
	prompt = strings.ReplaceAll(prompt, "{{title}}", meta.Title)
	prompt = strings.ReplaceAll(prompt, "{{uploader}}", meta.Uploader)
	prompt = strings.ReplaceAll(prompt, "{{date}}", meta.Date)
	prompt = strings.ReplaceAll(prompt, "{{duration}}", meta.Duration)
	prompt = strings.ReplaceAll(prompt, "{{resolution}}", meta.Resolution)
	prompt = strings.ReplaceAll(prompt, "{{format}}", meta.Format)
	prompt = strings.ReplaceAll(prompt, "{{size}}", meta.Size)
	prompt = strings.ReplaceAll(prompt, "{{description}}", meta.Description)
	prompt = strings.ReplaceAll(prompt, "{{subtitles}}", meta.Subtitles)
	prompt = strings.ReplaceAll(prompt, "{{subtitle_stats}}", meta.SubtitleStats)
	prompt = strings.ReplaceAll(prompt, "{{energy_candidates}}", meta.EnergyCandidates)
	prompt = strings.ReplaceAll(prompt, "{{language}}", settings.Language)

	if settings.AI.Prompt != "" {
		prompt = prompt + "\n\n" + settings.AI.Prompt
	}

	var content string
	var err error
	switch settings.AI.Provider {
	case "openai", "local", "custom", "deepseek", "siliconflow":
		content, err = m.callOpenAI(settings.AI, prompt, true)
	case "anthropic":
		content, err = m.callAnthropic(settings.AI, prompt)
	default:
		content, err = m.callOpenAI(settings.AI, prompt, true)
	}
	if err != nil {
		log.Printf("[Analysis] Error calling AI provider: %v", err)
		return nil, err
	}

	content = sanitizeJSONContent(content)

	var analysis AnalysisResult
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		log.Printf("[Analysis] Error unmarshalling analysis response: %v", err)
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
