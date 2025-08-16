package models

import "github.com/google/uuid"

type Review struct {
	BaseModel
	ReviewedByID uuid.UUID `gorm:"type:uuid;not null" json:"reviewed_by_id" valid:"required~Reviewed by ID is required"`
	ReviewedBy   User      `gorm:"foreignKey:ReviewedByID" json:"reviewed_by"`
	TeamID       uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	Team         Team      `gorm:"foreignKey:TeamID" json:"team"`
	RoundID      uuid.UUID `gorm:"type:uuid;not null" json:"round_id" valid:"required~Round ID is required"`
	Round        Round     `gorm:"foreignKey:RoundID" json:"round"`
	Scores       []Score   `gorm:"foreignKey:ReviewID" json:"scores"`
}
