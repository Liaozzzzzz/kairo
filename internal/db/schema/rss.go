package schema

type RSSItemStatus int

const (
	RSSItemStatusNew        RSSItemStatus = 0
	RSSItemStatusRead       RSSItemStatus = 1
	RSSItemStatusQueued     RSSItemStatus = 2
	RSSItemStatusFailed     RSSItemStatus = 3
	RSSItemStatusDownloaded RSSItemStatus = 4
)

type Feed struct {
	ID               string `gorm:"primaryKey;size:36" json:"id"`
	URL              string `gorm:"index" json:"url"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Thumbnail        string `json:"thumbnail"`
	LastUpdated      int64  `json:"last_updated"`
	UnreadCount      int    `json:"unread_count"`
	CustomDir        string `json:"custom_dir"`
	DownloadLatest   bool   `json:"download_latest"`
	Filters          string `json:"filters"`
	Tags             string `json:"tags"`
	FilenameTemplate string `json:"filename_template"`
	CategoryID       string `gorm:"index" json:"category_id"`
	Enabled          bool   `gorm:"index" json:"enabled"`
	CreatedAt        int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        int64  `gorm:"autoUpdateTime" json:"updated_at"`
}

type FeedItem struct {
	ID          string        `gorm:"primaryKey;size:36" json:"id"`
	FeedID      string        `gorm:"index;uniqueIndex:idx_feed_items_link" json:"feed_id"`
	Title       string        `json:"title"`
	Link        string        `gorm:"uniqueIndex:idx_feed_items_link" json:"link"`
	Description string        `json:"description"`
	PubDate     int64         `gorm:"index" json:"pub_date"`
	Status      RSSItemStatus `gorm:"index" json:"status"`
	Thumbnail   string        `json:"thumbnail"`
	CreatedAt   int64         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   int64         `gorm:"autoUpdateTime" json:"updated_at"`
}

type AddRSSTaskInput struct {
	FeedURL       string `json:"feed_url"`
	FeedTitle     string `json:"feed_title"`
	FeedThumbnail string `json:"feed_thumbnail"`
	ItemURL       string `json:"item_url"`
	ItemTitle     string `json:"item_title"`
	ItemThumbnail string `json:"item_thumbnail"`
	Dir           string `json:"dir"`
	CategoryID    string `json:"category_id"`
}

type AddRSSFeedInput struct {
	URL              string `json:"url"`
	CustomDir        string `json:"custom_dir"`
	DownloadLatest   bool   `json:"download_latest"`
	Filters          string `json:"filters"`
	Tags             string `json:"tags"`
	FilenameTemplate string `json:"filename_template"`
	CategoryID       string `json:"category_id"`
}
