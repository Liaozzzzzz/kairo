package schema

import "gorm.io/gorm"

type PublishType string

const (
	PublishTypeAuto   PublishType = "auto"
	PublishTypeManual PublishType = "manual"
)

type PublishStatus string

const (
	PublishStatusPending    PublishStatus = "pending"
	PublishStatusPublishing PublishStatus = "publishing"
	PublishStatusPublished  PublishStatus = "published"
	PublishStatusFailed     PublishStatus = "failed"
	PublishStatusCancelled  PublishStatus = "cancelled"
)

type PublishTask struct {
	ID              string        `gorm:"primaryKey;size:36" json:"id"`
	HighlightID     string        `gorm:"index" json:"highlight_id"`
	AccountID       string        `gorm:"index" json:"account_id"`
	Status          PublishStatus `gorm:"index" json:"status"`
	Type            PublishType   `json:"type"`
	ScheduledAt     int64         `gorm:"index" json:"scheduled_at"`
	PublishedAt     int64         `json:"published_at"`
	ErrorMessage    string        `gorm:"type:text" json:"error_message"`
	PlatformVideoID string        `json:"platform_video_id"`
	Title           string        `json:"title"`
	Description     string        `gorm:"type:text" json:"description"`
	Tags            string        `gorm:"type:text" json:"tags"`
	CreatedAt       int64         `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt       int64         `gorm:"autoUpdateTime:milli" json:"updated_at"`

	TagsList []string `gorm:"-" json:"tags_list"`

	Highlight *VideoHighlight  `gorm:"foreignKey:HighlightID" json:"highlight,omitempty"`
	Account   *PublishAccount  `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Platform  *PublishPlatform `gorm:"-" json:"platform,omitempty"`
}

type PublishTaskListResponse struct {
	Total int64         `json:"total"`
	Data  []PublishTask `json:"data"`
}

type CreatePublishTaskRequest struct {
	HighlightID string      `json:"highlight_id"`
	AccountID   string      `json:"account_id"`
	PublishType PublishType `json:"type"`
	ScheduledAt int64       `json:"scheduled_at"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Tags        string      `json:"tags"`
}

type UpdatePublishTaskScheduleRequest struct {
	ID          string `json:"id"`
	ScheduledAt int64  `json:"scheduled_at"`
}

type UpdatePublishTaskRequest struct {
	ID          string `json:"id"`
	ScheduledAt int64  `json:"scheduled_at"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

func (v *PublishTask) AfterFind(tx *gorm.DB) (err error) {
	if v.Account != nil {
		v.Platform = v.Account.Platform
	}
	return
}
