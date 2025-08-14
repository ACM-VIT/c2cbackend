package models

import "github.com/google/uuid"

type Score struct {
	BaseModel
	Design         int       `json:"design"`
	Implementation int       `json:"implementation"`
	Uniqueness     int       `json:"uniqueness"`
	Practicality   int       `json:"practicality"`
	Comments       *string   `gorm:"type:text" json:"comments,omitempty"`
	ReviewID       uuid.UUID `gorm:"type:uuid;not null" json:"review_id" valid:"required~Review ID is required"`
	Review         Review    `gorm:"foreignKey:ReviewID" json:"review"`
}
