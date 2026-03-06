package schema

type PublishPlatformStatus string

const (
	PublishPlatformStatusEnabled  PublishPlatformStatus = "enabled"
	PublishPlatformStatusDisabled PublishPlatformStatus = "disabled"
)

type Platform string

const (
	PlatformXiaohongshu Platform = "xiaohongshu"
	PlatformDouyin      Platform = "douyin"
)

type PublishPlatform struct {
	ID          string                `gorm:"primaryKey;size:36" json:"id"`
	Name        string                `gorm:"uniqueIndex" json:"name"`
	DisplayName string                `json:"display_name"`
	Status      PublishPlatformStatus `json:"status"`
	CreatedAt   int64                 `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64                 `gorm:"autoUpdateTime:milli" json:"updated_at"`
}
