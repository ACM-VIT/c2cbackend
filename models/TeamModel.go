package models

import "github.com/google/uuid"

type Team struct {
	BaseModel
	Name        string    `gorm:"type:varchar(100);not null" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphanumeric"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	Code        string    `gorm:"type:varchar(50);unique" json:"code"`
	GithubURL   string    `gorm:"type:varchar(255);unique" json:"github_url,omitempty"`
	FigmaURL    string    `gorm:"type:varchar(255);unique" json:"figma_url,omitempty"`
	Other       string    `gorm:"type:varchar(255);unique" json:"other,omitempty"`
	TrackID     uuid.UUID `gorm:"not null" json:"track_id"`
	Track       Track     `gorm:"foreignKey:TrackID" json:"track"`
	Users       []User    `json:"users"`
	Rounds      []Round   `gorm:"many2many:round_teams;" json:"rounds"`
	Reviews     []Review  `gorm:"foreignKey:TeamID" json:"reviews"`
}

type TeamJoinSchema struct {
	Code string `json:"code"`
}
