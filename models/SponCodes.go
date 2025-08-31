package models

import (
	"time"

	"github.com/google/uuid"
)

type SponCode struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Code        string     `gorm:"type:varchar(50);unique;not null" json:"code"`
	TeamID      *uuid.UUID `gorm:"type:uuid;unique;not null" json:"team_id"`
	Team        Team       `gorm:"foreignKey:TeamID" json:"team"`
	RequestedAt time.Time  `json:"requested_at"`
}
