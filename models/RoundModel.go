package models

import "time"

type Round struct {
	BaseModel
	Name        string    `gorm:"type:varchar(100);not null" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphanumeric"`
	RoundNumber int       `gorm:"type:int;not null;unique" json:"round_number" valid:"required~Round number is required"`
	StartTime   time.Time `gorm:"type:timestamp" json:"start_time,omitempty"`
	EndTime     time.Time `gorm:"type:timestamp" json:"end_time,omitempty"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	Teams       []Team    `gorm:"many2many:round_teams;" json:"teams"`
	Reviews     []Review  `gorm:"foreignKey:RoundID" json:"reviews"`
}
