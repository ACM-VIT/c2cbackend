package helpers

type CreateReviewReq struct {
	Design         int     `json:"design"`
	Implementation int     `json:"implementation"`
	Uniqueness     int     `json:"uniqueness"`
	Practicality   int     `json:"practicality"`
	Comments       *string `json:"comments,omitempty"`
}
