package teamcontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"

	"github.com/gofiber/fiber/v2"
)

func GetTeam(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var team models.Team
	if err := initializer.Database.Db.
		Preload("Users").
		First(&team, "id = ?", user.TeamID).
		Error; err != nil {
		return &fiber.Error{Code: 500, Message: "Failed to retrieve team"}
	}

	return c.JSON(fiber.Map{
		"message": "Team retrieved successfully",
		"team":    team,
	})
}

func GetAllTeams(c *fiber.Ctx) error {
	var teams []models.Team
	if err := initializer.Database.Db.
		Preload("Users").
		Find(&teams).Error; err != nil {
		return &fiber.Error{Code: 500, Message: "Failed to retrieve teams"}
	}

	return c.JSON(fiber.Map{
		"message": "Teams retrieved successfully",
		"teams":   teams,
	})
}
