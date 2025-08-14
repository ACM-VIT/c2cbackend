package roundcontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeamRanking struct {
	TeamID   uuid.UUID `json:"team_id"`
	TeamName string    `json:"team_name"`
	Total    int       `json:"total_score"`
	Rank     int       `json:"rank"`
}

func CreateRound(c *fiber.Ctx) error {
	var round models.Round
	if err := c.BodyParser(&round); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if round.RoundNumber == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Round number is required",
		})
	}

	var existingRound models.Round
	if err := initializer.Database.Db.Where("round_number = ?", round.RoundNumber).First(&existingRound).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Round with this number already exists",
		})
	}

	user := c.Locals("user").(models.User)

	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can create rounds",
		})
	}

	if err := initializer.Database.Db.Create(&round).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create round",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Round created successfully",
		"round":   round,
	})
}

func DeleteRound(c *fiber.Ctx) error {
	roundnumber := c.Params("rno")

	user := c.Locals("user").(models.User)

	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can delete rounds",
		})
	}

	var round models.Round
	if err := initializer.Database.Db.Where("round_number = ?", roundnumber).First(&round).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Round not found",
		})
	}

	if err := initializer.Database.Db.Delete(&round).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete round",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Round deleted successfully",
	})
}

func UpdateRound(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can update rounds",
		})
	}

	roundnumber := c.Params("rno")
	var round models.Round
	if err := initializer.Database.Db.Where("round_number = ?", roundnumber).First(&round).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Round not found",
		})
	}

	if err := c.BodyParser(&round); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := initializer.Database.Db.Save(&round).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update round",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Round updated successfully",
		"round":   round,
	})
}

func GetRoundTeamRankings(c *fiber.Ctx) error {
	rnoStr := c.Params("rno")
	rno, err := strconv.Atoi(rnoStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid round number"})
	}

	var round models.Round
	if err := initializer.Database.Db.Where("round_number = ?", rno).First(&round).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "round not found"})
	}

	type row struct {
		TeamID   uuid.UUID
		TeamName string
		Total    *int
	}

	var rows []row
	const q = `
SELECT
  t.id   AS team_id,
  t.name AS team_name,
  COALESCE(SUM(sd.design + sd.implementation + sd.uniqueness + sd.practicality), 0) AS total
FROM teams t
JOIN round_teams rt ON rt.team_id = t.id AND rt.round_id = $1
LEFT JOIN reviews r ON r.team_id = t.id AND r.round_id = $1
LEFT JOIN scores s ON s.review_id = r.id
LEFT JOIN score_details sd ON sd.score_id = s.id
GROUP BY t.id, t.name
`
	if err := initializer.Database.Db.Raw(q, round.ID).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch team rankings"})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		ti, tj := 0, 0
		if rows[i].Total != nil {
			ti = *rows[i].Total
		}
		if rows[j].Total != nil {
			tj = *rows[j].Total
		}
		if ti == tj {
			return rows[i].TeamName < rows[j].TeamName
		}
		return ti > tj
	})

	rankings := make([]TeamRanking, len(rows))
	var prevScore *int
	prevRank := 0
	for i, r := range rows {
		score := 0
		if r.Total != nil {
			score = *r.Total
		}
		if prevScore == nil || score != *prevScore {
			prevRank = i + 1
			prevScore = &score
		}
		rankings[i] = TeamRanking{
			TeamID:   r.TeamID,
			TeamName: r.TeamName,
			Total:    score,
			Rank:     prevRank,
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"round_number": rno,
		"round_id":     round.ID,
		"team_count":   len(rankings),
		"rankings":     rankings,
	})

}

func PromoteToRound(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admins can promote teams to the next round",
		})
	}

	rno := c.Params("rno")
	roundNumber, err := strconv.Atoi(rno)
	if err != nil || roundNumber < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid round number",
		})
	}

	var body helpers.PromoteReq
	if err := c.BodyParser(&body); err != nil || len(body.TeamIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provide non-empty team_ids array",
		})
	}

	db := initializer.Database.Db

	resp := helpers.PromoteResponse{
		CurrentRoundNo: roundNumber,
		NextRoundNo:    roundNumber + 1,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		var current models.Round
		if err := tx.Where("round_number = ?", roundNumber).First(&current).Error; err != nil {
			return fiber.ErrNotFound
		}
		resp.CurrentRoundID = current.ID

		var next models.Round
		err := tx.Where("round_number = ?", resp.NextRoundNo).First(&next).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				next = models.Round{
					Name:        fmt.Sprintf("Round %d", resp.NextRoundNo),
					RoundNumber: resp.NextRoundNo,
				}
				if err := tx.Create(&next).Error; err != nil {
					return err
				}
				resp.NextRoundCreated = true
			} else {
				return err
			}
		}
		resp.NextRoundID = next.ID

		var existingTeams []struct {
			ID uuid.UUID
		}
		if err := tx.Table("teams").
			Select("id").
			Where("id IN ?", body.TeamIDs).
			Scan(&existingTeams).Error; err != nil {
			return err
		}
		exists := make(map[uuid.UUID]struct{}, len(existingTeams))
		for _, t := range existingTeams {
			exists[t.ID] = struct{}{}
		}
		for _, id := range body.TeamIDs {
			if _, ok := exists[id]; !ok {
				resp.NotFound = append(resp.NotFound, id)
			}
		}

		var inCurrent []uuid.UUID
		if err := tx.Table("round_teams").
			Where("round_id = ? AND team_id IN ?", current.ID, body.TeamIDs).
			Pluck("team_id", &inCurrent).Error; err != nil {
			return err
		}
		inCurrentSet := make(map[uuid.UUID]struct{}, len(inCurrent))
		for _, id := range inCurrent {
			inCurrentSet[id] = struct{}{}
		}
		for _, id := range body.TeamIDs {
			if _, ok := exists[id]; ok {
				if _, ok2 := inCurrentSet[id]; !ok2 {
					resp.NotInCurrent = append(resp.NotInCurrent, id)
				}
			}
		}

		if len(inCurrent) == 0 {
			return nil
		}

		var already []uuid.UUID
		if err := tx.Table("round_teams").
			Where("round_id = ? AND team_id IN ?", next.ID, inCurrent).
			Pluck("team_id", &already).Error; err != nil {
			return err
		}
		alreadySet := make(map[uuid.UUID]struct{}, len(already))
		for _, id := range already {
			alreadySet[id] = struct{}{}
		}

		type rt struct {
			RoundID uuid.UUID `gorm:"column:round_id"`
			TeamID  uuid.UUID `gorm:"column:team_id"`
		}
		var rows []rt
		for _, id := range inCurrent {
			if _, ok := alreadySet[id]; ok {
				continue
			}
			rows = append(rows, rt{RoundID: next.ID, TeamID: id})
		}
		if len(rows) > 0 {
			if err := tx.Table("round_teams").
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&rows).Error; err != nil {
				return err
			}
		}

		resp.AlreadyInNext = append(resp.AlreadyInNext, already...)
		for _, id := range inCurrent {
			if _, ok := alreadySet[id]; !ok {
				resp.Promoted = append(resp.Promoted, id)
			}
		}

		return nil
	})

	if err != nil {
		if err == fiber.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Round not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to promote teams"})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// UNTESTED VIBE CODED FUNCTION
func AssignAllToRound(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can assign teams to rounds"})
	}

	roundNumber, err := strconv.Atoi(c.Params("rno"))
	if err != nil || roundNumber < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid round number"})
	}

	db := initializer.Database.Db

	var round models.Round
	var assigned int64

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// 1) Load the round
		if err := tx.Where("round_number = ?", roundNumber).First(&round).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}

		// 2) Clear existing links for this round
		if err := tx.Exec(`DELETE FROM round_teams WHERE round_id = ?`, round.ID).Error; err != nil {
			return err
		}

		// 3) Insert links for ALL current (non–soft-deleted) teams in one shot
		//    Adjust the deleted_at filter if you don't use gorm.DeletedAt
		res := tx.Exec(`
			INSERT INTO round_teams (round_id, team_id)
			SELECT ?, t.id
			FROM teams t
			WHERE t.deleted_at IS NULL
		`, round.ID)
		if res.Error != nil {
			return res.Error
		}
		assigned = res.RowsAffected // Postgres returns the inserted row count here
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "round not found"})
		}
		// If you STILL see the FK error here, verify your join table column names and FKs.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to assign teams to round"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"round_id":     round.ID,
		"round_number": roundNumber,
		"assigned":     assigned,
	})
}
