package video

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
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
	start, err := parseTimestampToSeconds(startFields[0])
	if err != nil {
		return 0, 0, false
	}
	end, err := parseTimestampToSeconds(endFields[0])
	if err != nil {
		return 0, 0, false
	}
	return start, end, true
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
		b.WriteString(formatTimestamp(seg.Start, true))
		b.WriteString(" --> ")
		b.WriteString(formatTimestamp(seg.End, true))
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

type subtitleStats struct {
	SegmentCount       int
	SubtitleDuration   float64
	TotalChars         int
	CharsPerMinute     float64
	AvgCharsPerSecond  float64
	AvgSegmentDuration float64
	SpeakingRatio      float64
	TopTokens          []string
}

type energyCandidate struct {
	Start  float64
	End    float64
	Score  float64
	Reason string
}

type energyWindow struct {
	Start       float64
	End         float64
	RawScore    float64
	TextLen     int
	Duration    float64
	PunctCount  int
	KeywordHits int
}

func buildSubtitleAnalysis(segments []subtitleSegment, videoDuration float64) (string, []energyCandidate) {
	if len(segments) == 0 {
		return "", nil
	}
	stats := computeSubtitleStats(segments, videoDuration)
	statsText := formatSubtitleStats(stats)
	candidates := detectEnergyCandidates(segments, videoDuration)
	return statsText, candidates
}

func computeSubtitleStats(segments []subtitleSegment, videoDuration float64) subtitleStats {
	var totalDuration float64
	var totalChars int
	tokenCounts := map[string]int{}
	tokenRegex := regexp.MustCompile(`[\p{Han}]+|[A-Za-z0-9]+`)

	for _, seg := range segments {
		if seg.End <= seg.Start {
			continue
		}
		duration := seg.End - seg.Start
		totalDuration += duration
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		totalChars += len([]rune(text))
		tokens := tokenRegex.FindAllString(text, -1)
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if containsHan(token) {
				if len([]rune(token)) < 2 {
					continue
				}
			} else {
				token = strings.ToLower(token)
				if len(token) < 3 {
					continue
				}
			}
			tokenCounts[token]++
		}
	}

	if videoDuration <= 0 {
		videoDuration = maxSegmentEnd(segments)
	}
	avgCharsPerSecond := 0.0
	charsPerMinute := 0.0
	avgSegmentDuration := 0.0
	speakingRatio := 0.0
	if totalDuration > 0 {
		avgCharsPerSecond = float64(totalChars) / totalDuration
		charsPerMinute = avgCharsPerSecond * 60
		avgSegmentDuration = totalDuration / float64(len(segments))
	}
	if videoDuration > 0 {
		speakingRatio = totalDuration / videoDuration
	}

	topTokens := topTokenList(tokenCounts, 10)

	return subtitleStats{
		SegmentCount:       len(segments),
		SubtitleDuration:   totalDuration,
		TotalChars:         totalChars,
		CharsPerMinute:     charsPerMinute,
		AvgCharsPerSecond:  avgCharsPerSecond,
		AvgSegmentDuration: avgSegmentDuration,
		SpeakingRatio:      speakingRatio,
		TopTokens:          topTokens,
	}
}

func detectEnergyCandidates(segments []subtitleSegment, videoDuration float64) []energyCandidate {
	maxTime := maxSegmentEnd(segments)
	if videoDuration > 0 && videoDuration > maxTime {
		maxTime = videoDuration
	}
	if maxTime <= 0 {
		return nil
	}

	windowSize := 45.0
	step := 15.0
	var windows []energyWindow
	keywords := highEnergyKeywords()

	for start := 0.0; start < maxTime; start += step {
		end := start + windowSize
		if end > maxTime {
			end = maxTime
		}
		window := buildEnergyWindow(segments, start, end, keywords)
		if window.Duration < windowSize*0.4 || window.TextLen == 0 {
			continue
		}
		speed := float64(window.TextLen) / math.Max(window.Duration, 1)
		window.RawScore = speed*0.5 + float64(window.PunctCount)*0.5 + float64(window.KeywordHits)*1.2
		windows = append(windows, window)
	}

	if len(windows) == 0 {
		return nil
	}

	minScore := windows[0].RawScore
	maxScore := windows[0].RawScore
	for _, w := range windows[1:] {
		if w.RawScore < minScore {
			minScore = w.RawScore
		}
		if w.RawScore > maxScore {
			maxScore = w.RawScore
		}
	}

	type scoredCandidate struct {
		Start  float64
		End    float64
		Score  float64
		Reason string
	}

	var candidates []scoredCandidate
	for _, w := range windows {
		score := 0.5
		if maxScore-minScore > 0.0001 {
			score = (w.RawScore - minScore) / (maxScore - minScore)
		}
		speed := float64(w.TextLen) / math.Max(w.Duration, 1)
		reason := fmt.Sprintf("speech_rate=%.1f chars/s, punct=%d, keywords=%d", speed, w.PunctCount, w.KeywordHits)
		candidates = append(candidates, scoredCandidate{
			Start:  w.Start,
			End:    w.End,
			Score:  score,
			Reason: reason,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Start < candidates[j].Start
		}
		return candidates[i].Score > candidates[j].Score
	})

	var selected []energyCandidate
	for _, candidate := range candidates {
		if len(selected) >= 5 {
			break
		}
		if overlapsSelected(candidate.Start, candidate.End, selected) {
			continue
		}
		selected = append(selected, energyCandidate{
			Start:  candidate.Start,
			End:    candidate.End,
			Score:  candidate.Score,
			Reason: candidate.Reason,
		})
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Start < selected[j].Start
	})

	return expandEnergyCandidates(selected, segments, maxTime)
}

func buildEnergyWindow(segments []subtitleSegment, start float64, end float64, keywords []string) energyWindow {
	var textLen int
	var duration float64
	var punctCount int
	var keywordHits int
	for _, seg := range segments {
		if seg.End <= start || seg.Start >= end {
			continue
		}
		overlap := math.Min(seg.End, end) - math.Max(seg.Start, start)
		if overlap <= 0 {
			continue
		}
		duration += overlap
		textLen += len([]rune(seg.Text))
		punctCount += countPunctuation(seg.Text)
		keywordHits += countKeywordHits(seg.Text, keywords)
	}
	return energyWindow{
		Start:       start,
		End:         end,
		TextLen:     textLen,
		Duration:    duration,
		PunctCount:  punctCount,
		KeywordHits: keywordHits,
	}
}

func formatSubtitleStats(stats subtitleStats) string {
	topTokens := "n/a"
	if len(stats.TopTokens) > 0 {
		topTokens = strings.Join(stats.TopTokens, ", ")
	}
	return fmt.Sprintf("segments=%d; subtitle_duration=%.1fs; speaking_ratio=%.2f; chars_per_min=%.1f; avg_segment=%.1fs; top_keywords=%s",
		stats.SegmentCount,
		stats.SubtitleDuration,
		stats.SpeakingRatio,
		stats.CharsPerMinute,
		stats.AvgSegmentDuration,
		topTokens,
	)
}

func formatEnergyCandidates(candidates []energyCandidate) string {
	if len(candidates) == 0 {
		return ""
	}
	var b strings.Builder
	for i, candidate := range candidates {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%d. %s-%s score=%.2f %s",
			i+1,
			formatTimestamp(candidate.Start, false),
			formatTimestamp(candidate.End, false),
			candidate.Score,
			candidate.Reason,
		)
	}
	return b.String()
}

func countPunctuation(text string) int {
	count := strings.Count(text, "!")
	count += strings.Count(text, "！")
	count += strings.Count(text, "?")
	count += strings.Count(text, "？")
	return count
}

func countKeywordHits(text string, keywords []string) int {
	if text == "" || len(keywords) == 0 {
		return 0
	}
	lower := strings.ToLower(text)
	total := 0
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		total += strings.Count(lower, keyword)
	}
	return total
}

func highEnergyKeywords() []string {
	return []string{
		"高能", "爆点", "高潮", "反转", "冲突", "震撼", "惊讶", "不可思议", "炸裂", "燃", "激动",
		"笑死", "爆笑", "崩溃", "尖叫", "关键", "重大", "最强", "绝了", "太强", "太猛", "太棒",
		"wow", "amazing", "insane", "unbelievable", "crazy", "shocking", "incredible", "epic", "hilarious",
	}
}

func topTokenList(tokenCounts map[string]int, limit int) []string {
	type kv struct {
		Token string
		Count int
	}
	var items []kv
	for token, count := range tokenCounts {
		items = append(items, kv{Token: token, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Token < items[j].Token
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Token)
	}
	return result
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func maxSegmentEnd(segments []subtitleSegment) float64 {
	maxEnd := 0.0
	for _, seg := range segments {
		if seg.End > maxEnd {
			maxEnd = seg.End
		}
	}
	return maxEnd
}

func overlapsSelected(start float64, end float64, selected []energyCandidate) bool {
	for _, candidate := range selected {
		overlap := math.Min(end, candidate.End) - math.Max(start, candidate.Start)
		if overlap <= 0 {
			continue
		}
		minDur := math.Min(end-start, candidate.End-candidate.Start)
		if minDur <= 0 {
			continue
		}
		if overlap/minDur > 0.5 {
			return true
		}
	}
	return false
}

func expandEnergyCandidates(candidates []energyCandidate, segments []subtitleSegment, maxTime float64) []energyCandidate {
	minDuration := 60.0
	targetDuration := 90.0
	maxDuration := 150.0
	expanded := make([]energyCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		start := candidate.Start
		end := candidate.End
		duration := end - start
		if duration < minDuration {
			mid := (start + end) / 2
			start = mid - targetDuration/2
			end = mid + targetDuration/2
		} else if duration > maxDuration {
			mid := (start + end) / 2
			start = mid - maxDuration/2
			end = mid + maxDuration/2
		}
		start, end = clampRange(start, end, maxTime)
		start, end = snapRangeToSegments(segments, start, end)
		expanded = append(expanded, energyCandidate{
			Start:  start,
			End:    end,
			Score:  candidate.Score,
			Reason: candidate.Reason,
		})
	}
	return expanded
}

func clampRange(start float64, end float64, maxTime float64) (float64, float64) {
	if maxTime > 0 {
		if start < 0 {
			start = 0
		}
		if end > maxTime {
			end = maxTime
		}
	}
	if end < start {
		end = start
	}
	return start, end
}

func snapRangeToSegments(segments []subtitleSegment, start float64, end float64) (float64, float64) {
	if len(segments) == 0 {
		return start, end
	}
	snappedStart := start
	for _, seg := range segments {
		if seg.Start <= start {
			snappedStart = seg.Start
		} else {
			break
		}
	}
	snappedEnd := end
	for _, seg := range segments {
		if seg.End >= end {
			snappedEnd = seg.End
			break
		}
	}
	if snappedEnd < snappedStart {
		return start, end
	}
	return snappedStart, snappedEnd
}

func parseTimestampToSeconds(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("invalid timestamp: %s", raw)
	}
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

func formatTimestamp(seconds float64, includeMillis bool) string {
	if seconds < 0 {
		seconds = 0
	}
	if includeMillis {
		totalMillis := int64(seconds*1000 + 0.5)
		hours := totalMillis / 3600000
		minutes := (totalMillis % 3600000) / 60000
		secs := (totalMillis % 60000) / 1000
		millis := totalMillis % 1000
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, secs, millis)
	}
	totalSeconds := int64(seconds + 0.5)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}
