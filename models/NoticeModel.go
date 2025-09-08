package models

type Notice struct {
	BaseModel
	Information string `gorm:"type:varchar(100);not null;unique" json:"information" valid:"required~Information is required"`
}
