package models

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

type DownloadFile struct {
	Path      string  `json:"path"`
	Size      string  `json:"size"`
	SizeBytes int64   `json:"size_bytes"`
	Progress  float64 `json:"progress"`
}

type PlaylistItem struct {
	Index     int     `json:"index"`
	Title     string  `json:"title"`
	Duration  float64 `json:"duration"`
	Thumbnail string  `json:"thumbnail"`
	URL       string  `json:"url"`
}

type DownloadTask struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Dir         string     `json:"dir"`
	Quality     string     `json:"quality"` // "best", "1080p", "720p", "audio"
	Format      string     `json:"format"`  // "original", "webm", "mp4", "mkv", "avi", "flv", "mov"
	FormatID    string     `json:"format_id"`
	ParentID    string     `json:"parent_id"`
	SourceType  SourceType `json:"source_type"`
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
	FilePath    string     `json:"file_path"`
	TotalBytes  int64      `json:"total_bytes"`
	TrimStart   string     `json:"trim_start"`
	TrimEnd     string     `json:"trim_end"`
	TrimMode    TrimMode   `json:"trim_mode"`
	CreatedAt   int64      `json:"created_at"`
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
	Title         string          `json:"title"`
	Thumbnail     string          `json:"thumbnail"`
	Duration      float64         `json:"duration"`
	Qualities     []QualityOption `json:"qualities"`
	SourceType    SourceType      `json:"source_type"`
	PlaylistItems []PlaylistItem  `json:"playlist_items"`
	TotalItems    int             `json:"total_items"`
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
}

type AddRSSTaskInput struct {
	FeedURL       string `json:"feed_url"`
	FeedTitle     string `json:"feed_title"`
	FeedThumbnail string `json:"feed_thumbnail"`
	ItemURL       string `json:"item_url"`
	ItemTitle     string `json:"item_title"`
	ItemThumbnail string `json:"item_thumbnail"`
	Dir           string `json:"dir"`
}

type RSSFeed struct {
	ID               string `json:"id"`
	URL              string `json:"url"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Thumbnail        string `json:"thumbnail"`
	LastUpdated      int64  `json:"last_updated"`
	UnreadCount      int    `json:"unread_count"`
	CustomDir        string `json:"custom_dir"`
	DownloadLatest   bool   `json:"download_latest"`
	Filters          string `json:"filters"` // Comma separated keywords
	Tags             string `json:"tags"`
	FilenameTemplate string `json:"filename_template"`
	Enabled          bool   `json:"enabled"`
}

type AddRSSFeedInput struct {
	URL              string `json:"url"`
	CustomDir        string `json:"custom_dir"`
	DownloadLatest   bool   `json:"download_latest"`
	Filters          string `json:"filters"`
	Tags             string `json:"tags"`
	FilenameTemplate string `json:"filename_template"`
}

type RSSItemStatus int

const (
	RSSItemStatusNew        RSSItemStatus = 0
	RSSItemStatusRead       RSSItemStatus = 1
	RSSItemStatusQueued     RSSItemStatus = 2
	RSSItemStatusFailed     RSSItemStatus = 3
	RSSItemStatusDownloaded RSSItemStatus = 4
)

type RSSItem struct {
	ID          string        `json:"id"` // GUID or URL
	FeedID      string        `json:"feed_id"`
	Title       string        `json:"title"`
	Link        string        `json:"link"`
	Description string        `json:"description"`
	PubDate     int64         `json:"pub_date"`
	Status      RSSItemStatus `json:"status"`
	Thumbnail   string        `json:"thumbnail"`
}

type AddPlaylistTaskInput struct {
	URL           string         `json:"url"`
	Dir           string         `json:"dir"`
	Title         string         `json:"title"`
	Thumbnail     string         `json:"thumbnail"`
	PlaylistItems []PlaylistItem `json:"playlist_items"`
}
