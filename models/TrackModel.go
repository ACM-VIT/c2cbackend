package models

type Track struct {
	BaseModel
	Title       string `gorm:"type:varchar(100);not null" json:"title" valid:"required~Title is required,matches(^[a-zA-Z0-9 ]+$)~Title must be alphanumeric"`
	Description string `gorm:"type:text" json:"description,omitempty"`
}
