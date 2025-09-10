package detailcontroller

import (
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"

	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AddRoom(c *fiber.Ctx) error {
	// Read authenticated user from context
	uLoc := c.Locals("user")
	if uLoc == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	// accept both pointer and value
	switch v := uLoc.(type) {
	case models.User:
		user = v
	case *models.User:
		if v == nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		user = *v
	default:
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	if !user.Internal {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Only internal participants can add room details"})
	}

	type reqBody struct {
		Room  string `json:"room_number"`
		Block string `json:"block"`
	}
	var rb reqBody
	if err := c.BodyParser(&rb); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body", "details": err.Error()})
	}

	// Basic validation: room and block lengths
	if rb.Room == "" && rb.Block == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "room_number or block must be provided"})
	}

	// Reload user from DB to ensure fresh copy and to get primary key
	if err := initializer.Database.Db.Where("email = ?", user.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	if !user.Internal {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Only internal participants can add room details"})
	}

	user.SetRoomBlock(rb.Room, rb.Block)
	if err := initializer.Database.Db.Save(&user).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update room details"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Room details updated", "user": user})
}
