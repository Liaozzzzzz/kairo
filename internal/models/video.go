package models

type Video struct {
	ID         string  `json:"id"`
	TaskID     string  `json:"task_id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	FilePath   string  `json:"file_path"`
	Thumbnail  string  `json:"thumbnail"`
	Duration   float64 `json:"duration"`
	Size       int64   `json:"size"`
	Format     string  `json:"format"`
	Resolution string  `json:"resolution"`
	CreatedAt  int64   `json:"created_at"`

	// Metadata
	Description string   `json:"description"`
	Uploader    string   `json:"uploader"`
	Subtitles   []string `json:"subtitles"` // Paths to subtitle files

	// AI Analysis
	Summary    string        `json:"summary"`
	Tags       []string      `json:"tags"`
	Evaluation string        `json:"evaluation"`
	Highlights []AIHighlight `json:"highlights"`
	Status     string        `json:"status"` // "pending", "processing", "completed", "failed", "none"
}

type AIHighlight struct {
	ID          string `json:"id"`
	VideoID     string `json:"video_id"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
	FilePath    string `json:"file_path,omitempty"`
}

type VideoFilter struct {
	Status string `json:"status"` // "all", "analyzed", "unanalyzed"
	Query  string `json:"query"`
}
