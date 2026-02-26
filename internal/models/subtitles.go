package models

type SubtitleStatus int

const (
	SubtitleStatusGenerating SubtitleStatus = iota
	SubtitleStatusSuccess
	SubtitleStatusFailed
	SubtitleStatusPending
)

type SubtitleSource int

const (
	SubtitleSourceBuiltin SubtitleSource = iota
	SubtitleSourceASR
	SubtitleSourceManual
	SubtitleSourceTranslation
)

type VideoSubtitle struct {
	ID        string         `json:"id"`
	VideoID   string         `json:"video_id"`
	FilePath  string         `json:"file_path"`
	Language  string         `json:"language"`
	Status    SubtitleStatus `json:"status"`
	Source    SubtitleSource `json:"source"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

type TranslateSubtitleInput struct {
	VideoID        string `json:"video_id"`
	SubtitleID     string `json:"subtitle_id"`
	TargetLanguage string `json:"target_language"`
}
