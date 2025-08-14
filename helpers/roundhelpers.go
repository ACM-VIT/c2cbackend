package helpers

import "github.com/google/uuid"

type PromoteReq struct {
	TeamIDs []uuid.UUID `json:"team_ids"`
}
type PromoteResponse struct {
	CurrentRoundID   uuid.UUID   `json:"current_round_id"`
	CurrentRoundNo   int         `json:"current_round_number"`
	NextRoundID      uuid.UUID   `json:"next_round_id"`
	NextRoundNo      int         `json:"next_round_number"`
	NextRoundCreated bool        `json:"next_round_created"`
	Promoted         []uuid.UUID `json:"promoted"`
	AlreadyInNext    []uuid.UUID `json:"already_in_next"`
	NotInCurrent     []uuid.UUID `json:"not_in_current_round"`
	NotFound         []uuid.UUID `json:"not_found"`
}

type TeamRanking struct {
	TeamID   uuid.UUID `json:"team_id"`
	TeamName string    `json:"team_name"`
	Total    int       `json:"total_score"`
	Rank     int       `json:"rank"`
}
