package ai

import (
	"strings"

	"Kairo/internal/config"
)

type VideoMetadata struct {
	Title       string
	Description string
	Subtitles   string
	Uploader    string
	Duration    string
	Resolution  string
	Format      string
	Size        string
	Date        string
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
	cfg := config.GetSettings().AI
	if !cfg.Enabled {
		return nil, ErrAIDisabled
	}

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = `Analyze the following video content.
Provide a JSON response with the following fields:
- "summary": A concise summary of the video content (max 200 words).
- "tags": A list of 5-10 relevant tags/keywords.
- "evaluation": A brief evaluation of the video quality/value.
- "highlights": A list of 3-5 key moments/highlights. Each item must have:
  - "start": Start time in HH:MM:SS format (e.g. "00:01:30").
  - "end": End time in HH:MM:SS format.
  - "description": Brief description of the highlight.

Video Information:
- Title: {{title}}
- Uploader: {{uploader}}
- Date: {{date}}
- Duration: {{duration}}
- Resolution: {{resolution}}
- Format: {{format}}
- Size: {{size}}

Video Description:
{{description}}

Video Subtitles (excerpt):
{{subtitles}}`
	}

	prompt = strings.ReplaceAll(prompt, "{{title}}", meta.Title)
	prompt = strings.ReplaceAll(prompt, "{{uploader}}", meta.Uploader)
	prompt = strings.ReplaceAll(prompt, "{{date}}", meta.Date)
	prompt = strings.ReplaceAll(prompt, "{{duration}}", meta.Duration)
	prompt = strings.ReplaceAll(prompt, "{{resolution}}", meta.Resolution)
	prompt = strings.ReplaceAll(prompt, "{{format}}", meta.Format)
	prompt = strings.ReplaceAll(prompt, "{{size}}", meta.Size)
	prompt = strings.ReplaceAll(prompt, "{{description}}", meta.Description)
	prompt = strings.ReplaceAll(prompt, "{{subtitles}}", meta.Subtitles)

	switch cfg.Provider {
	case "openai", "local", "custom", "deepseek", "siliconflow":
		return m.callOpenAI(cfg, prompt)
	case "anthropic":
		return m.callAnthropic(cfg, prompt)
	default:
		return m.callOpenAI(cfg, prompt)
	}
}
