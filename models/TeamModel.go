package models

type Team struct {
	BaseModel
	Name        string   `gorm:"type:varchar(100);not null" json:"name" valid:"required~Name is required,matches(^[a-zA-Z0-9 ]+$)~Name must be alphanumeric"`
	Description *string  `gorm:"type:text" json:"description,omitempty"`
	Code        string   `gorm:"type:varchar(50);unique" json:"code"`
	Users       []User   `json:"users"`
	Rounds      []Round  `gorm:"many2many:round_teams;" json:"rounds"`
	Reviews     []Review `gorm:"foreignKey:TeamID" json:"reviews"`
}

type TeamJoinSchema struct {
	Code string `json:"code"`
}
