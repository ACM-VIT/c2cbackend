package models

import "github.com/google/uuid"

type Score struct {
	BaseModel
	InnovationRelevance         int       `json:"innovation_relevance"`
	TechnicalDepthComplexity    int       `json:"technical_depth_complexity"`
	ImplementationFunctionality int       `json:"implementation_functionality"`
	UserExperiencePresentation  int       `json:"user_experience_presentation"`
	ProgressDevelopment         int       `json:"progress_development"`
	Comments                    *string   `gorm:"type:text" json:"comments,omitempty"`
	ReviewID                    uuid.UUID `gorm:"type:uuid;not null" json:"review_id" valid:"required~Review ID is required"`
	Review                      Review    `gorm:"foreignKey:ReviewID" json:"review"`
}
