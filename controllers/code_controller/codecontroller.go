package codecontroller

import (
	"errors"
	"os"
	"strings"
	"time"

	"c2cbackend/initializer"
	"c2cbackend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func availableCodes() []string {
	raw := os.Getenv("SPON_CODES")
	if strings.TrimSpace(raw) == "" {
		return []string{"ALPHA", "BRAVO", "CHARLIE", "DELTA", "ECHO"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func RequestCode(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if user.TeamID == nil || *user.TeamID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "user is not part of any team")
	}
	teamID := *user.TeamID

	db := initializer.Database.Db

	var existing models.SponCode
	if err := db.Where("team_id = ?", teamID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":         existing.Code,
			"team_id":      existing.TeamID,
			"requested_at": existing.RequestedAt,
		})
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var assigned models.SponCode
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("team_id = ?", teamID).
			First(&existing).Error; err == nil {
			assigned = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var usedRows []models.SponCode
		if err := tx.Select("code").Find(&usedRows).Error; err != nil {
			return err
		}
		used := make(map[string]struct{}, len(usedRows))
		for _, r := range usedRows {
			used[r.Code] = struct{}{}
		}

		var chosen string
		for _, cand := range availableCodes() {
			if _, taken := used[cand]; !taken {
				chosen = cand
				break
			}
		}
		if chosen == "" {
			return fiber.NewError(fiber.StatusConflict, "no codes left to assign")
		}

		row := models.SponCode{
			ID:          uuid.New(),
			Code:        chosen,
			TeamID:      teamID,
			RequestedAt: time.Now(),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}

		assigned = row
		return nil
	}); err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":         assigned.Code,
		"team_id":      assigned.TeamID,
		"requested_at": assigned.RequestedAt,
	})
}
