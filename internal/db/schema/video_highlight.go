package schema

type VideoHighlight struct {
	ID          string `gorm:"primaryKey;size:36" json:"id"`
	VideoID     string `gorm:"index" json:"video_id"`
	StartTime   string `gorm:"column:start_time" json:"start"` // Mapped to 'start' for frontend compatibility
	EndTime     string `gorm:"column:end_time" json:"end"`     // Mapped to 'end' for frontend compatibility
	Title       string `json:"title"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	CreatedAt   int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   int64  `gorm:"autoUpdateTime" json:"updated_at"`
}
