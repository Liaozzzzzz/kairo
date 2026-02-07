package models

type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusStarting    TaskStatus = "starting"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusMerging     TaskStatus = "merging"
	TaskStatusPaused      TaskStatus = "paused"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusError       TaskStatus = "error"
)

type DownloadTask struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Dir         string     `json:"dir"`
	Quality     string     `json:"quality"` // "best", "1080p", "720p", "audio"
	Format      string     `json:"format"`  // "original", "webm", "mp4", "mkv", "avi", "flv", "mov"
	Filename    string     `json:"filename"`
	Status      TaskStatus `json:"status"`
	Progress    float64    `json:"progress"`
	Title       string     `json:"title"`
	Thumbnail   string     `json:"thumbnail"`
	TotalSize   string     `json:"total_size"`
	Speed       string     `json:"speed"`
	Eta         string     `json:"eta"`
	CurrentItem int        `json:"current_item"`
	TotalItems  int        `json:"total_items"`
	LogPath     string     `json:"log_path"`
	FileExists  bool       `json:"file_exists"`
	TotalBytes  int64      `json:"total_bytes"`
}

type QualityOption struct {
	Label      string `json:"label"`
	Value      string `json:"value"`
	VideoSize  string `json:"video_size"`
	AudioSize  string `json:"audio_size"`
	TotalSize  string `json:"total_size"`
	VideoBytes int64  `json:"video_bytes"`
	AudioBytes int64  `json:"audio_bytes"`
	TotalBytes int64  `json:"total_bytes"`
}

type VideoInfo struct {
	Title     string          `json:"title"`
	Thumbnail string          `json:"thumbnail"`
	Duration  float64         `json:"duration"`
	Qualities []QualityOption `json:"qualities"`
}

type AddTaskInput struct {
	URL        string `json:"url"`
	Quality    string `json:"quality"`
	Format     string `json:"format"`
	Dir        string `json:"dir"`
	Title      string `json:"title"`
	Thumbnail  string `json:"thumbnail"`
	TotalBytes int64  `json:"total_bytes"`
}
