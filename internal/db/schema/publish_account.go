package schema

type PublishAccountStatus string

const (
	PublishAccountStatusUnknown PublishAccountStatus = "unknown"
	PublishAccountStatusActive  PublishAccountStatus = "active"
	PublishAccountStatusInvalid PublishAccountStatus = "invalid"
)

type PublishAccount struct {
	ID              string               `gorm:"primaryKey;size:36" json:"id"`
	PlatformID      string               `gorm:"index" json:"platform_id"`
	Name            string               `json:"name"`
	CookiePath      string               `json:"cookie_path"`
	PublishInterval string               `json:"publish_interval"` // e.g. "1h", "30m"
	IsEnabled       bool                 `gorm:"default:true" json:"is_enabled"`
	Status          PublishAccountStatus `gorm:"index" json:"status"`
	LastChecked     int64                `json:"last_checked"`
	CreatedAt       int64                `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt       int64                `gorm:"autoUpdateTime:milli" json:"updated_at"`
	Platform        *PublishPlatform     `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
}
