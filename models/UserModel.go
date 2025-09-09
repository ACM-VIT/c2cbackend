package models

import "github.com/google/uuid"

type User struct {
	BaseModel
	Name              string     `gorm:"type:varchar(100);not null" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphabetic"`
	Email             string     `gorm:"type:varchar(100);not null;unique" json:"email" valid:"required~Email is required,email~Email is not valid"`
	ProfilePictureURL string     `gorm:"type:text" json:"profile_picture_url" valid:"url~URL is not valid"`
	ContactNumber     string     `gorm:"type:varchar(20);not null" json:"contact_number" valid:"required~Contact number is required,numeric~Contact number must be numeric"`
	Gender            string     `gorm:"type:varchar(10);not null" json:"gender" valid:"in(male|female|other)~Gender must be male female or other"`
	RegNo             *string    `gorm:"type:varchar(20);unique" json:"reg_no,omitempty"`
	Internal          bool       `gorm:"default:false" json:"internal"`
	Hosteller         bool       `gorm:"default:false" json:"hosteller"`
	CollegeName       string     `gorm:"type:text;" json:"college_name"`
	Role              UserRole   `gorm:"type:varchar(20);not null" json:"role" valid:"in(admin|reviewer|participant)~Role must be admin/reviewer/participant"`
	TeamID            *uuid.UUID `gorm:"type:uuid;" json:"team_id"`
	Team              *Team      `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	CheckedIn         bool       `gorm:"default:false" json:"checked_in,omitempty"`
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
