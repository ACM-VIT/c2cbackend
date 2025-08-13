package models

import "github.com/google/uuid"

type ScoreDetail struct {
	BaseModel
	Design         int     `json:"design"`
	Implementation int     `json:"implementation"`
	Uniqueness     int     `json:"uniqueness"`
	Practicality   int     `json:"practicality"`
	Comments       *string `gorm:"type:text" json:"comments,omitempty"`

	ScoreID uuid.UUID `gorm:"type:uuid;not null" json:"score_id" valid:"required~Score ID is required"`
	Score   Score     `gorm:"foreignKey:ScoreID" json:"score"`
}
