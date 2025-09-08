package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/google/uuid"
)

// add: custom type that marshals to/from JSON for storing an array of strings
type StringArray []string

// Value implements driver.Valuer so GORM can store StringArray as JSON
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(sa))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner so GORM can read JSON into StringArray
func (sa *StringArray) Scan(src interface{}) error {
	if src == nil {
		*sa = StringArray{}
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, (*[]string)(sa))
	case string:
		return json.Unmarshal([]byte(s), (*[]string)(sa))
	default:
		// fallback: try to marshal then unmarshal
		b, err := json.Marshal(s)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, (*[]string)(sa))
	}
}

type Team struct {
	BaseModel
	Name                 string      `gorm:"type:varchar(100);not null;unique" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphanumeric"`
	Description          *string     `gorm:"type:text" json:"description,omitempty"`
	Code                 string      `gorm:"type:varchar(50);unique" json:"code"`
	GithubURL            string      `gorm:"type:varchar(255);" json:"github_url,omitempty"`
	FigmaURL             string      `gorm:"type:varchar(255);" json:"figma_url,omitempty"`
	TechStack            StringArray `gorm:"type:json" json:"tech_stack,omitempty"`
	Other                string      `gorm:"type:varchar(255);" json:"other,omitempty"`
	GitHubInstallationID string      `json:"github_installation_id"`
	TrackID              *uuid.UUID  `json:"track_id"`
	Track                Track       `gorm:"foreignKey:TrackID" json:"track"`
	Users                []User      `json:"users"`
	Rounds               []Round     `gorm:"many2many:round_teams;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"rounds"`
	Reviews              []Review    `gorm:"foreignKey:TeamID" json:"reviews"`
}

type TeamJoinSchema struct {
	Code string `json:"code"`
}
