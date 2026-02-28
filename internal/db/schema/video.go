package schema

import (
	"encoding/json"
	"strings"
)

type Video struct {
	ID          string `gorm:"primaryKey;size:36" json:"id"`
	TaskID      string `gorm:"index" json:"task_id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	FilePath    string `json:"file_path"`
	Thumbnail   string `json:"thumbnail"`
	Duration    float64 `json:"duration"`
	Size        int64   `json:"size"`
	Format      string  `json:"format"`
	Resolution  string  `json:"resolution"`
	CreatedAt   int64 `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   int64 `gorm:"autoUpdateTime" json:"updated_at"`
	Description string `json:"description"`
	Uploader    string `json:"uploader"`
	Summary     string `json:"summary"`
	Tags        string `gorm:"type:text" json:"-"`
	Evaluation  string `json:"evaluation"`
	CategoryID  string `gorm:"index" json:"category_id"`
	Status      string `gorm:"index" json:"status"`

	// Virtual fields for JSON
	TagsList   []string         `gorm:"-" json:"tags"`
	Highlights []VideoHighlight `gorm:"foreignKey:VideoID" json:"highlights"`
}

// AfterFind hook to parse Tags string to TagsList slice
func (v *Video) AfterFind() (err error) {
	if v.Tags != "" {
		if strings.HasPrefix(v.Tags, "[") {
			_ = json.Unmarshal([]byte(v.Tags), &v.TagsList)
		} else {
			// Fallback for comma separated string if any
			v.TagsList = strings.Split(v.Tags, ",")
		}
	} else {
		v.TagsList = []string{}
	}
	return
}

type VideoHighlight struct {
	ID          string `gorm:"primaryKey;size:36" json:"id"`
	VideoID     string `gorm:"index" json:"video_id"`
	StartTime   string `gorm:"column:start_time" json:"start"` // Mapped to 'start' for frontend compatibility
	EndTime     string `gorm:"column:end_time" json:"end"`     // Mapped to 'end' for frontend compatibility
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	CreatedAt   int64 `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   int64 `gorm:"autoUpdateTime" json:"updated_at"`
}

type PlaylistItem struct {
	Index     int     `json:"index"`
	Title     string  `json:"title"`
	Duration  float64 `json:"duration"`
	Thumbnail string  `json:"thumbnail"`
	URL       string  `json:"url"`
}

type QualityOption struct {
	Label      string `json:"label"`
	Value      string `json:"value"`
	FormatID   string `json:"format_id"`
	VideoSize  string `json:"video_size"`
	AudioSize  string `json:"audio_size"`
	TotalSize  string `json:"total_size"`
	VideoBytes int64  `json:"video_bytes"`
	AudioBytes int64  `json:"audio_bytes"`
	TotalBytes int64  `json:"total_bytes"`
}

type VideoInfo struct {
	Title         string            `json:"title"`
	Thumbnail     string            `json:"thumbnail"`
	Duration      float64           `json:"duration"`
	Qualities     []QualityOption   `json:"qualities"`
	SourceType    SourceType        `json:"source_type"`
	PlaylistItems []PlaylistItem    `json:"playlist_items"`
	TotalItems    int               `json:"total_items"`
}

type VideoFilter struct {
	Status string `json:"status"` // "all", "analyzed", "unanalyzed"
	Query  string `json:"query"`
}
