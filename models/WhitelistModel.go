package models

type Whitelist struct {
	BaseModel
	Email    string `gorm:"type:varchar(100);not null;unique" json:"email" valid:"required~Email is required,email~Invalid email format"`
	Internal bool   `gorm:"default:false" json:"internal"`
}
