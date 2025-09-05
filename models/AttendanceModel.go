package models

import (
	"time"

	"github.com/google/uuid"
)

type Attendance struct {
	BaseModel
	UserID    uuid.UUID `gorm:"not null" json:"user_id" valid:"required~User ID is required"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user"`
	Timestamp time.Time `gorm:"not null" json:"timestamp" valid:"required~Timestamp is required"`
}
