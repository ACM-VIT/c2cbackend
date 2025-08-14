package models

import "github.com/google/uuid"

type User struct {
	BaseModel
	Name              string     `gorm:"type:varchar(100);not null" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphabetic"`
	Email             string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"email" valid:"required~Email is required,email~Email is not valid"`
	ProfilePictureURL string     `gorm:"type:text" json:"profile_picture_url" valid:"url~URL is not valid"`
	ContactNumber     string     `gorm:"type:varchar(20);not null" json:"contact_number" valid:"required~Contact number is required,numeric~Contact number must be numeric"`
	Gender            string     `gorm:"type:varchar(10);not null" json:"gender" valid:"in(male|female|other)~Gender must be male female or other"`
	RegNo             string     `gorm:"type:varchar(20);not null;uniqueIndex" json:"reg_no" valid:"required~Registration number is required"`
	Role              UserRole   `gorm:"type:varchar(20);not null" json:"role" valid:"in(admin|reviewer|participant)~Role must be admin/reviewer/participant"`
	TeamID            *uuid.UUID `gorm:"type:uuid;" json:"team_id"`
	Team              *Team      `gorm:"foreignKey:TeamID" json:"team,omitempty"`
}

type UserRole string

const (
	RoleAdmin       UserRole = "admin"
	RoleReviewer    UserRole = "reviewer"
	RoleParticipant UserRole = "participant"
)

func IsValidRole(role UserRole) bool {
	switch role {
	case RoleAdmin, RoleReviewer, RoleParticipant:
		return true
	default:
		return false
	}
}
