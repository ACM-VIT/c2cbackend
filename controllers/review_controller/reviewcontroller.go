package reviewcontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func PostReview(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	rno, err := strconv.Atoi(c.Params("rno"))
	if err != nil || rno < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid round number"})
	}

	teamID, err := uuid.Parse(c.Params("team_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid team id"})
	}

	var body helpers.CreateReviewReq
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	db := initializer.Database.Db

	var review models.Review
	var score models.Score

	if txErr := db.Transaction(func(tx *gorm.DB) error {
		// round
		var round models.Round
		if err := tx.Where("round_number = ?", rno).First(&round).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helpers.ErrRoundNotFound
			}
			return err
		}

		// team
		var team models.Team
		if err := tx.First(&team, "id = ?", teamID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helpers.ErrTeamNotFound
			}
			return err
		}

		// team must be in the round
		var cnt int64
		if err := tx.Table("round_teams").
			Where("round_id = ? AND team_id = ?", round.ID, team.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return helpers.ErrTeamNotInRound
		}

		// user must not have already reviewed this team in this round
		var exists int64
		if err := tx.Model(&models.Review{}).
			Where("reviewed_by_id = ? AND team_id = ? AND round_id = ?", user.ID, team.ID, round.ID).
			Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return helpers.ErrDuplicateReview
		}

		// create review
		review = models.Review{
			ReviewedByID: user.ID,
			TeamID:       team.ID,
			RoundID:      round.ID,
		}
		if err := tx.Create(&review).Error; err != nil {
			return err
		}

		// create score with NEW fields
		score = models.Score{
			ReviewID:                    review.ID,
			InnovationRelevance:         body.InnovationRelevance,
			TechnicalDepthComplexity:    body.TechnicalDepthComplexity,
			ImplementationFunctionality: body.ImplementationFunctionality,
			UserExperiencePresentation:  body.UserExperiencePresentation,
			ProgressDevelopment:         body.ProgressDevelopment,
			Comments:                    body.Comments,
		}
		if err := tx.Create(&score).Error; err != nil {
			return err
		}

		return nil
	}); txErr != nil {
		switch {
		case errors.Is(txErr, helpers.ErrRoundNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "round not found"})
		case errors.Is(txErr, helpers.ErrTeamNotFound):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid team id"})
		case errors.Is(txErr, helpers.ErrTeamNotInRound):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team not in this round"})
		case errors.Is(txErr, helpers.ErrDuplicateReview):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "you already reviewed this team for this round"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create review"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"review_id":      review.ID,
		"score_id":       score.ID,
		"round_id":       review.RoundID,
		"team_id":        review.TeamID,
		"reviewed_by_id": review.ReviewedByID,
		"detail": fiber.Map{
			"innovation_relevance":         score.InnovationRelevance,
			"technical_depth_complexity":   score.TechnicalDepthComplexity,
			"implementation_functionality": score.ImplementationFunctionality,
			"user_experience_presentation": score.UserExperiencePresentation,
			"progress_development":         score.ProgressDevelopment,
			"comments":                     score.Comments,
		},
	})
}

func DeleteReview(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can delete reviews"})
	}

	// Parse round number
	rno, err := strconv.Atoi(c.Params("rno"))
	if err != nil || rno < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid round number"})
	}

	// Parse team UUID
	teamID, err := uuid.Parse(c.Params("team_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid team id"})
	}

	db := initializer.Database.Db

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Find the round by round_number
		var round models.Round
		if err := tx.Where("round_number = ?", rno).First(&round).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "round not found")
			}
			return err
		}

		// Find the review for that round and team
		var review models.Review
		if err := tx.Where("round_id = ? AND team_id = ?", round.ID, teamID).First(&review).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "review not found")
			}
			return err
		}

		// Delete scores (optional if ON DELETE CASCADE is set on FK)
		if err := tx.Where("review_id = ?", review.ID).Delete(&models.Score{}).Error; err != nil {
			return err
		}

		// Delete the review
		if err := tx.Delete(&review).Error; err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		var ferr *fiber.Error
		if errors.As(txErr, &ferr) {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete review"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "review deleted successfully"})
}

func GetReviews(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can view reviews"})
	}

	var reviews []models.Review
	if err := initializer.Database.Db.
		Preload("ReviewedBy").
		Preload("Team").
		Preload("Round").
		Preload("Scores").
		Find(&reviews).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to retrieve reviews"})
	}

	return c.Status(fiber.StatusOK).JSON(reviews)
}
