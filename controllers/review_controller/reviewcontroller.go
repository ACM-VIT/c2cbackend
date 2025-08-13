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
	var detail models.ScoreDetail

	err = db.Transaction(func(tx *gorm.DB) error {
		var round models.Round
		if err := tx.Where("round_number = ?", rno).First(&round).Error; err != nil {
			return fiber.ErrNotFound
		}

		var team models.Team
		if err := tx.Where("id = ?", teamID).First(&team).Error; err != nil {
			return fiber.ErrBadRequest
		}

		var cnt int64
		if err := tx.Table("round_teams").
			Where("round_id = ? AND team_id = ?", round.ID, team.ID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return fiber.ErrBadRequest
		}

		var exists int64
		if err := tx.Model(&models.Review{}).
			Where("reviewed_by_id = ? AND team_id = ? AND round_id = ?", user.ID, team.ID, round.ID).
			Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return fiber.ErrConflict
		}

		review = models.Review{
			ReviewedByID: user.ID,
			TeamID:       team.ID,
			RoundID:      round.ID,
		}
		if err := tx.Create(&review).Error; err != nil {
			return err
		}

		score = models.Score{
			ReviewID: review.ID,
		}
		if err := tx.Create(&score).Error; err != nil {
			return err
		}

		detail = models.ScoreDetail{
			ScoreID:        score.ID,
			Design:         body.Design,
			Implementation: body.Implementation,
			Uniqueness:     body.Uniqueness,
			Practicality:   body.Practicality,
			Comments:       body.Comments,
		}
		if err := tx.Create(&detail).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		switch err {
		case fiber.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "round not found"})
		case fiber.ErrBadRequest:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team not in this round or invalid"})
		case fiber.ErrConflict:
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
			"design":         detail.Design,
			"implementation": detail.Implementation,
			"uniqueness":     detail.Uniqueness,
			"practicality":   detail.Practicality,
			"comments":       detail.Comments,
		},
	})
}

func DeleteReview(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != models.RoleAdmin {
		return &fiber.Error{Code: 403, Message: "Only admins can delete reviews"}
	}

	reviewID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return &fiber.Error{Code: 400, Message: "Invalid review id"}
	}

	db := initializer.Database.Db

	if err := db.Transaction(func(tx *gorm.DB) error {
		var review models.Review
		if err := tx.Where("id = ?", reviewID).First(&review).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &fiber.Error{Code: 404, Message: "Review not found"}
			}
			return &fiber.Error{Code: 500, Message: "Failed to load review"}
		}

		// Delete all Scores for this review (ScoreDetail has OnDelete:CASCADE to Score)
		if err := tx.Where("review_id = ?", review.ID).Delete(&models.Score{}).Error; err != nil {
			return &fiber.Error{Code: 500, Message: "Failed to delete linked scores"}
		}

		// Finally delete the review
		if err := tx.Delete(&review).Error; err != nil {
			return &fiber.Error{Code: 500, Message: "Failed to delete review"}
		}
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return fe
		}
		return &fiber.Error{Code: 500, Message: "Failed to delete review"}
	}

	return c.Status(200).JSON(fiber.Map{"message": "Review deleted successfully"})
}

// YE NAHI CHAL RAHA LIKHNE ME AALAS PLEASE WILL DO LATER
func GetReviews(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	if user.Role != models.RoleAdmin {
		return &fiber.Error{Code: 403, Message: "Only admins can view reviews"}
	}

	var reviews []models.Review
	if err := initializer.Database.Db.Find(&reviews).Error; err != nil {
		return &fiber.Error{Code: 500, Message: "Failed to retrieve reviews"}
	}

	return c.Status(200).JSON(reviews)
}
