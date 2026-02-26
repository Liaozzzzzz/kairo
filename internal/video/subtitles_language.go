package video

import (
	"Kairo/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

type subtitleOutputEntry struct {
	Path     string
	Language string
}

func extractSubtitleEntriesFromOutput(output string) []subtitleOutputEntry {
	var entries []subtitleOutputEntry
	seen := map[string]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "subtitle") {
			continue
		}
		idx := strings.LastIndex(lower, "to:")
		if idx == -1 {
			continue
		}
		candidate := strings.TrimSpace(line[idx+3:])
		candidate = strings.Trim(candidate, "\"'")
		candidate = strings.TrimRight(candidate, "\r")
		if candidate == "" || !isSubtitleFile(candidate) {
			continue
		}
		if _, statErr := os.Stat(candidate); statErr != nil {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		entries = append(entries, subtitleOutputEntry{
			Path:     candidate,
			Language: detectLanguageFromSubtitlePath(candidate),
		})
	}
	return entries
}

func isSubtitleFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".vtt" || ext == ".srt"
}

func detectLanguageFromSubtitlePath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if base == "" {
		return detectLanguageFromSubtitleContent(path)
	}
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return detectLanguageFromSubtitleContent(path)
	}
	return parts[len(parts)-1]
}

func detectLanguageFromSubtitleContent(path string) string {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return utils.DetectLanguageFromText(utils.ExtractTextFromVTT(string(contentBytes)))
}
