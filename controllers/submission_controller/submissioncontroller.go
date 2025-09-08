package submissioncontroller

import (
    "c2cbackend/initializer"
    "c2cbackend/models"
    "net/http"

    "github.com/gofiber/fiber/v2"
)

// Admin and Reviewer can view all submissions
func GetAll(c *fiber.Ctx) error {
    u, ok := c.Locals("user").(models.User)
    if !ok || (u.Role != models.RoleAdmin && u.Role != models.RoleReviewer) {
        return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
    }
    var subs []models.Submission
    if err := initializer.Database.Db.Preload("Team").Preload("Round").Find(&subs).Error; err != nil {
        return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch submissions"})
    }
    return c.Status(http.StatusOK).JSON(fiber.Map{"submissions": subs})
}

