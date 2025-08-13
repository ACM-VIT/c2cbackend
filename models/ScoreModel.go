package models

import "github.com/google/uuid"

type Score struct {
	BaseModel
	ReviewID uuid.UUID `gorm:"type:uuid;not null" json:"review_id" valid:"required~Review ID is required"`
	Review   Review    `gorm:"foreignKey:ReviewID" json:"review"`

	Detail *ScoreDetail `gorm:"constraint:OnDelete:CASCADE;" json:"detail"`
}
