package helpers

type CreateReviewReq struct {
	InnovationRelevance         int     `json:"innovation_relevance"`
	TechnicalDepthComplexity    int     `json:"technical_depth_complexity"`
	ImplementationFunctionality int     `json:"implementation_functionality"`
	UserExperiencePresentation  int     `json:"user_experience_presentation"`
	ProgressDevelopment         int     `json:"progress_development"`
	Comments                    *string `json:"comments,omitempty"`
}
