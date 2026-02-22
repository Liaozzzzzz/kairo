package video

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type subtitleSegment struct {
	Start float64
	End   float64
	Text  string
}

func parseSubtitleFile(path string) ([]subtitleSegment, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(contentBytes)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("empty subtitle file")
	}

	splitter := regexp.MustCompile(`\n\s*\n`)
	blocks := splitter.Split(content, -1)
	var segments []subtitleSegment
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 2 {
			continue
		}
		var timestampLine string
		var textLines []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "-->") {
				timestampLine = line
				continue
			}
			if isSubtitleIndex(line) {
				continue
			}
			textLines = append(textLines, line)
		}
		if timestampLine == "" || len(textLines) == 0 {
			continue
		}
		start, end, ok := parseTimestampLine(timestampLine)
		if !ok {
			continue
		}
		text := strings.Join(textLines, " ")
		text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		segments = append(segments, subtitleSegment{
			Start: start,
			End:   end,
			Text:  text,
		})
	}
	return segments, nil
}

func parseTimestampLine(line string) (float64, float64, bool) {
	parts := strings.Split(line, "-->")
	if len(parts) < 2 {
		return 0, 0, false
	}
	startFields := strings.Fields(strings.TrimSpace(parts[0]))
	endFields := strings.Fields(strings.TrimSpace(parts[1]))
	if len(startFields) == 0 || len(endFields) == 0 {
		return 0, 0, false
	}
	start, err := parseSubtitleTimestamp(startFields[0])
	if err != nil {
		return 0, 0, false
	}
	end, err := parseSubtitleTimestamp(endFields[0])
	if err != nil {
		return 0, 0, false
	}
	return start, end, true
}

func parseSubtitleTimestamp(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, ",", ".")
	parts := strings.Split(s, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, fmt.Errorf("invalid timestamp: %s", raw)
	}
	hours := 0
	minutes := 0
	var secondsPart string
	if len(parts) == 2 {
		min, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minutes = min
		secondsPart = parts[1]
	} else {
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		min, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		hours = h
		minutes = min
		secondsPart = parts[2]
	}
	secs, err := strconv.ParseFloat(secondsPart, 64)
	if err != nil {
		return 0, err
	}
	return float64(hours*3600+minutes*60) + secs, nil
}

func formatSubtitleTimestamp(seconds float64) string {
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

func buildSubtitleText(segments []subtitleSegment) string {
	var b strings.Builder
	for i, seg := range segments {
		if seg.Text == "" {
			continue
		}
		if i > 0 && b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(formatSubtitleTimestamp(seg.Start))
		b.WriteString(" --> ")
		b.WriteString(formatSubtitleTimestamp(seg.End))
		b.WriteString("\n")
		b.WriteString(seg.Text)
	}
	return b.String()
}

func isSubtitleIndex(line string) bool {
	if line == "" {
		return false
	}
	_, err := strconv.Atoi(line)
	return err == nil
}
