package models

import "github.com/google/uuid"

type Submission struct {
	BaseModel
	PPTURL      string     `gorm:"type:varchar(255)" json:"ppt_url" valid:"required~PPT URL is required"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	RoundID     uuid.UUID  `gorm:"type:uuid;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"round_id"`
	TeamID      *uuid.UUID `json:"team_id"`
	Team        Team       `gorm:"foreignKey:TeamID" json:"team"`
}
