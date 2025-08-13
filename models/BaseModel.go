package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primary_key"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (base *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	base.ID = uuid
	return nil
}
