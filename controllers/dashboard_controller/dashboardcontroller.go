package usercontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Dashboard(c *fiber.Ctx) error {
	u := c.Locals("user")
	if u == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "User not in context"})
	}

	var email string
	switch v := u.(type) {
	case models.User:
		email = v.Email
	case *models.User:
		if v != nil {
			email = v.Email
		}
	}
	if email == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user in context"})
	}

	var user models.User
	if err := initializer.Database.Db.
		Preload("Team.Users").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"user":         user,
		"team_members": user.Team.Users,
	})
}
