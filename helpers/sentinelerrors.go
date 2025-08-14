package helpers

import "errors"

var (
	ErrRoundNotFound   = errors.New("round not found")
	ErrTeamNotFound    = errors.New("team not found")
	ErrTeamNotInRound  = errors.New("team not in this round")
	ErrDuplicateReview = errors.New("duplicate review")
	ErrReviewNotFound  = errors.New("review not found")
)
