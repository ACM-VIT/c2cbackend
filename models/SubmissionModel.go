package models

import "github.com/google/uuid"

type Submission struct {
	BaseModel
	PPTURL      string     `gorm:"type:text" json:"ppt_url" valid:"required~PPT URL is required"`
	Title       string     `gorm:"type:varchar(100);" json:"title" valid:"required~Title is required"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	RoundID     uuid.UUID  `gorm:"type:uuid;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;uniqueIndex:idx_team_round" json:"round_id"`
	TeamID      *uuid.UUID `gorm:"uniqueIndex:idx_team_round" json:"team_id"`
	Team        Team       `gorm:"foreignKey:TeamID" json:"team"`
}
