package attendancecontroller

import (
	"errors"
	"time"

	"c2cbackend/initializer"
	"c2cbackend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MarkAttendance(c *fiber.Ctx) error {
	admin, ok := c.Locals("user").(models.User)
	if !ok || (admin.Role != models.RoleAdmin && admin.Role != models.RoleReviewer) {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	user := models.User{}
	pathParameter := c.Params("user_id")
	if pathParameter != "" {
		uid, err := uuid.Parse(pathParameter)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid user ID")
		}
		db := initializer.Database.Db
		if err := db.First(&user, "id = ?", uid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "user not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch user")
		}
		
		attendance := models.Attendance{
			UserID:    user.ID,
			Timestamp: time.Now().UTC(),
		}
		
		if err := db.Create(&attendance).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to mark attendance")
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":    "Attendance marked successfully",
			"attendance": attendance,
		})
	} else {
		return fiber.NewError(fiber.StatusBadRequest, "user ID is required")
	}
}


func FilterAttendance(c *fiber.Ctx) error {
	admin, ok := c.Locals("user").(models.User)
	if !ok || (admin.Role != models.RoleAdmin && admin.Role != models.RoleReviewer) {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}

	db := initializer.Database.Db

	var users []models.User
	if err := db.Preload("Attendances").Find(&users).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch users")
	}

	var attendances []models.Attendance
	if err := db.Preload("User").Find(&attendances).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch attendances")
	}

	attendanceMap := make(map[uuid.UUID][]models.Attendance)
	for _, attendance := range attendances {
		attendanceMap[attendance.UserID] = append(attendanceMap[attendance.UserID], attendance)
	}

	userHasAttendance := make(map[uuid.UUID]bool)
	for uid, atts := range attendanceMap {
		userHasAttendance[uid] = len(atts) > 0
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Attendances fetched successfully",
		"users": userHasAttendance,
	})
}