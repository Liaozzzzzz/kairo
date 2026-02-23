package ai

import (
	_ "embed"
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

	prompt := loadPrompt(settings.AI)

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

	switch settings.AI.Provider {
	case "openai", "local", "custom", "deepseek", "siliconflow":
		return m.callOpenAI(settings.AI, prompt)
	case "anthropic":
		return m.callAnthropic(settings.AI, prompt)
	default:
		return m.callOpenAI(settings.AI, prompt)
	}
}

func loadPrompt(cfg config.AIConfig) string {
	basePrompt := strings.TrimSpace(defaultAIPrompt)
	extraPrompt := strings.TrimSpace(cfg.Prompt)
	if basePrompt == "" {
		return extraPrompt
	}
	if extraPrompt == "" {
		return basePrompt
	}
	return basePrompt + "\n\n" + extraPrompt
}
