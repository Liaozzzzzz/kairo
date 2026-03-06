package schema

type PublishRecord struct {
	ID        string        `gorm:"primaryKey;size:36" json:"id"`
	TaskID    string        `gorm:"index" json:"task_id"`
	Trigger   string        `json:"trigger"` // auto, manual
	Status    PublishStatus `gorm:"index" json:"status"`
	Message   string        `gorm:"type:text" json:"message"`
	CreatedAt int64         `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt int64         `gorm:"autoUpdateTime:milli" json:"updated_at"`
}
