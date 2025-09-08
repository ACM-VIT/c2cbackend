package noticecontroller

import (
	"c2cbackend/helpers"
	"c2cbackend/initializer"
	"c2cbackend/models"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)


func GetNotices(c *fiber.Ctx) error {
	db := initializer.Database.Db

	var notices []models.Notice
	if err := db.Find(&notices).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch notices"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"notices": notices})
}


func PostNotice(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	
	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	var body helpers.CreateNoticeReq
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	
	notice := models.Notice{
		Information: body.Information,
	}
	db := initializer.Database.Db
	if err := db.Create(&notice).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "notice already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create notice"})
	}
	
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"notice": notice})
}

func DeleteNotice(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(models.User)
	if !ok || user.ID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	
	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	
	noticeID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notice id"})
	}
	
	db := initializer.Database.Db
	if err := db.Delete(&models.Notice{}, "id = ?", noticeID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete notice"})
	}
	
	return c.SendStatus(fiber.StatusNoContent)
}
