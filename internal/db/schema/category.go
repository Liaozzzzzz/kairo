package schema

type CategorySource string

const (
	CategorySourceBuiltin CategorySource = "builtin"
	CategorySourceCustom  CategorySource = "custom"
)

type Category struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	Name      string         `gorm:"index" json:"name"`
	Prompt    string         `gorm:"type:text" json:"prompt"`
	Source    CategorySource `json:"source"`
	CreatedAt int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
}
