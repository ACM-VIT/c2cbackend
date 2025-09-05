package whitelistcontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func CheckWhitelist(c *fiber.Ctx) error {
	rawClaims := c.Locals("claims")
	if rawClaims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing auth claims"})
	}

	claims, ok := rawClaims.(map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid auth claims"})
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email not found in auth claims"})
	}

	var whitelistEntry models.Whitelist
	result := initializer.Database.Db.Where("email = ?", email).First(&whitelistEntry)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Email not whitelisted"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Email is whitelisted", "internal": whitelistEntry.Internal})
}
