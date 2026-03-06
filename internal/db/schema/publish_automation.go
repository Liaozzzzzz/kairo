package schema

import (
	"gorm.io/gorm"
)

type PublishAutomation struct {
	ID                  string `gorm:"primaryKey;size:36" json:"id"`
	CategoryID          string `gorm:"index" json:"category_id"` // Empty for all categories
	AccountID           string `gorm:"index" json:"account_id"`
	TitleTemplate       string `json:"title_template"` // e.g. "{{title}} - {{date}}"
	DescriptionTemplate string `gorm:"type:text" json:"description_template"`
	Tags                string `gorm:"type:text" json:"tags"` // Comma separated
	IsEnabled           bool   `gorm:"default:true" json:"is_enabled"`
	Cron                string `json:"cron"` // Cron expression for scheduling. Empty means manual approval.
	CreatedAt           int64  `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt           int64  `gorm:"autoUpdateTime:milli" json:"updated_at"`

	Platform *PublishPlatform `gorm:"-" json:"platform"`

	Category *Category       `gorm:"foreignKey:CategoryID;constraint:false" json:"category,omitempty"`
	Account  *PublishAccount `gorm:"foreignKey:AccountID;constraint:false" json:"account,omitempty"`
}

type CreatePublishAutomationRequest struct {
	CategoryID          string `json:"category_id"`
	AccountID           string `json:"account_id"`
	TitleTemplate       string `json:"title_template"`
	DescriptionTemplate string `json:"description_template"`
	Tags                string `json:"tags"`
	Cron                string `json:"cron"`
	IsEnabled           bool   `json:"is_enabled"`
}

type UpdatePublishAutomationRequest struct {
	ID                  string `json:"id"`
	TitleTemplate       string `json:"title_template"`
	DescriptionTemplate string `json:"description_template"`
	Tags                string `json:"tags"`
	Cron                string `json:"cron"`
	IsEnabled           bool   `json:"is_enabled"`
}

func (v *PublishAutomation) AfterFind(tx *gorm.DB) (err error) {
	if v.Account.Platform != nil {
		v.Platform = v.Account.Platform
		v.Account.Platform = nil
	}
	return
}
