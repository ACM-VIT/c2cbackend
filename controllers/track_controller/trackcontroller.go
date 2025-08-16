package trackcontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func CreateTrack(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	type CreateTrackInput struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	var input CreateTrackInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	track := models.Track{
		Title:       input.Title,
		Description: strings.TrimSpace(input.Description),
	}

	if err := initializer.Database.Db.Create(&track).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create track"})
	}

	return c.Status(fiber.StatusCreated).JSON(track)
}

func GetTracks(c *fiber.Ctx) error {
	var tracks []models.Track
	if err := initializer.Database.Db.Find(&tracks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve tracks"})
	}
	return c.Status(fiber.StatusOK).JSON(tracks)
}

func UpdateTrack(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	id := c.Params("trackid")

	var track models.Track
	if err := initializer.Database.Db.Where("id = ?", id).First(&track).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Track not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load track"})
	}

	type UpdateTrackInput struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	var input UpdateTrackInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Title != nil {
		newTitle := strings.TrimSpace(*input.Title)
		if newTitle == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title cannot be empty"})
		}
		track.Title = newTitle
	}
	if input.Description != nil {
		track.Description = strings.TrimSpace(*input.Description)
	}

	if err := initializer.Database.Db.Save(&track).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update track"})
	}

	return c.Status(fiber.StatusOK).JSON(track)
}

func DeleteTrack(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	id := c.Params("trackid")

	var track models.Track
	if err := initializer.Database.Db.Where("id = ?", id).First(&track).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Track not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load track"})
	}

	var teamCount int64
	if err := initializer.Database.Db.Model(&models.Team{}).
		Where("track_id = ?", track.ID).
		Count(&teamCount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify track usage"})
	}
	if teamCount > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot delete track with existing teams"})
	}

	if err := initializer.Database.Db.Delete(&track).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete track"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
