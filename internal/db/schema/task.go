package schema

type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusStarting    TaskStatus = "starting"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusMerging     TaskStatus = "merging"
	TaskStatusTrimming    TaskStatus = "trimming"
	TaskStatusPaused      TaskStatus = "paused"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusTrimFailed  TaskStatus = "trim_failed"
	TaskStatusError       TaskStatus = "error"
)

type SourceType int

const (
	SourceTypeSingle   SourceType = 0
	SourceTypePlaylist SourceType = 1
	SourceTypeRSS      SourceType = 2
)

type TrimMode string

const (
	TrimModeNone      TrimMode = "none"
	TrimModeOverwrite TrimMode = "overwrite"
	TrimModeKeep      TrimMode = "keep"
)

type Task struct {
	ID         string     `gorm:"primaryKey;size:36" json:"id"`
	URL        string     `gorm:"index" json:"url"`
	Dir        string     `json:"dir"`
	Quality    string     `json:"quality"`
	Format     string     `json:"format"`
	FormatID   string     `json:"format_id"`
	ParentID   string     `gorm:"index" json:"parent_id"`
	SourceType SourceType `gorm:"index" json:"source_type"`
	Status     TaskStatus `gorm:"index" json:"status"`
	Progress   float64    `json:"progress"`
	Title      string     `json:"title"`
	Thumbnail  string     `json:"thumbnail"`
	Speed      string     `json:"speed"`
	Eta        string     `json:"eta"`
	LogPath    string     `json:"log_path"`
	FileExists bool       `gorm:"column:file_exists" json:"file_exists"`
	FilePath   string     `json:"file_path"`
	TotalBytes int64      `json:"total_bytes"`
	TrimStart  string     `json:"trim_start"`
	TrimEnd    string     `json:"trim_end"`
	TrimMode   TrimMode   `json:"trim_mode"`
	CategoryID string     `gorm:"index" json:"category_id"`
	CreatedAt  int64      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  int64      `gorm:"autoUpdateTime" json:"updated_at"`
}

type DownloadFile struct {
	Path      string  `json:"path"`
	Size      string  `json:"size"`
	SizeBytes int64   `json:"size_bytes"`
	Progress  float64 `json:"progress"`
}

type AddTaskInput struct {
	URL        string     `json:"url"`
	Quality    string     `json:"quality"`
	Format     string     `json:"format"`
	FormatID   string     `json:"format_id"`
	Dir        string     `json:"dir"`
	Title      string     `json:"title"`
	Thumbnail  string     `json:"thumbnail"`
	TotalBytes int64      `json:"total_bytes"`
	TrimStart  string     `json:"trim_start"`
	TrimEnd    string     `json:"trim_end"`
	TrimMode   TrimMode   `json:"trim_mode"`
	SourceType SourceType `json:"source_type"`
	CategoryID string     `json:"category_id"`
}

type AddPlaylistTaskInput struct {
	URL           string         `json:"url"`
	Dir           string         `json:"dir"`
	Title         string         `json:"title"`
	Thumbnail     string         `json:"thumbnail"`
	PlaylistItems []PlaylistItem `json:"playlist_items"`
	CategoryID    string         `json:"category_id"`
}
