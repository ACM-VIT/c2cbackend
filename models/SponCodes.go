package models

import (
	"time"

	"github.com/google/uuid"
)

type SponCode struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Code        string     `gorm:"type:varchar(50);unique;not null" json:"code"`
	TeamID      *uuid.UUID `gorm:"type:uuid" json:"team_id"`
	Team        Team       `gorm:"foreignKey:TeamID" json:"team"`
	Status      ReqStatus  `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
}

type ReqStatus string

const (
	StatusPending  ReqStatus = "pending"
	StatusApproved ReqStatus = "approved"
	StatusDenied   ReqStatus = "denied"
)